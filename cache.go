package cache

import (
	"errors"
	"fmt"
	"sync"
	"time"

	keymanager "github.com/manhcuongincusar1/pointer-cache/key_manager"
)

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	ZeroExpiration time.Duration = 0

	// Pointer Size
	// If system 32 bit/8 = 4 bytes
	// If system 64 bit/8 = 8 bytes
	// Calculate Pointer size base on system type: return bytes
	PtrSize = (32 << uintptr(^uintptr(0)>>63)) / 8
)

// Cacher doesn't control dev's value
// All Value will be wrap into an Item
// Cacher store a map[string]*Item. A pointer only cost 8 bytes
// It is safe while the map's buckets will never shrink down if we do not matain a sharding method
type Item struct {
	Object     any
	Expiration int64
	Mem        int64
}

// Returns true if the item has expired.
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type cache struct {
	option     *Option
	items      map[string]*Item
	mu         sync.RWMutex
	onEvicted  func(string, any)
	janitor    *janitor
	memUsage   int64
	keyManager keymanager.KeyManager
}

// Alloc allows to expose used memory as bytes
func (p *cache) Alloc() int64 {
	return p.memUsage
}

// WithKeyManager allows consumer side (Developer) to add their own implement in developmet time
// Example: developer can add key manager base on other algorithm like LRU_cache or stack
// All need to do is implement keymanager.KeyManager
func (p *cache) WithKeyManager(manager keymanager.KeyManager) {
	p.keyManager = manager
}

// Add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (p *cache) Set(k string, v interface{}, d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Set data
	return p.set(k, v, d)
}

// Add an item to the cache, replacing any existing item, using the default
// expiration.
func (p *cache) SetDefault(k string, x interface{}) {
	p.Set(k, x, p.option.DefaultExpiration)
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (p *cache) Add(k string, x interface{}, d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, found := p.get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}

	return p.set(k, x, d)
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
func (p *cache) Delete(k string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, evicted := p.delete(k)
	if evicted {
		p.onEvicted(k, v)
	}
}

// Delete all expired items from the cache.
func (p *cache) DeleteExpired() {
	var evictedItems []keyAndValue
	now := time.Now().UnixNano()
	p.mu.Lock()
	for k, v := range p.items {
		// "Inlining" of expired
		if v.Expiration > 0 && now > v.Expiration {
			ov, evicted := p.delete(k)
			if evicted {
				evictedItems = append(evictedItems, keyAndValue{k, ov})
			}
		}
	}
	p.mu.Unlock()
	for _, v := range evictedItems {
		p.onEvicted(v.key, v.value)
	}
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (p *cache) Get(k string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// "Inlining" of get and Expired
	item, found := p.items[k]
	if !found {
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}

	return item.Object, true
}

// Set a new value for the cache key only if it already exists, and the existing
// item hasn't expired. Returns an error otherwise.
func (p *cache) Replace(k string, x interface{}, d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	item, found := p.getItem(k)
	if !found {
		return fmt.Errorf("Item %s doesn't exist", k)
	}

	// Deduct mem usage
	p.deductMemUsage(item.Mem)
	if err := p.set(k, x, d); err != nil {
		return err
	}

	// Bring the key to last of the queue
	p.keyManager.Delete(k)
	p.keyManager.Add(k)

	return nil
}

// GetWithExpiration returns an item and its expiration time from the cache.
// It returns the item or nil, the expiration time if one is set (if the item
// never expires a zero value for time.Time is returned), and a bool indicating
// whether the key was found.
func (p *cache) GetWithExpiration(k string) (interface{}, time.Time, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// "Inlining" of get and Expired
	item, found := p.items[k]
	if !found {
		return nil, time.Time{}, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, time.Time{}, false
		}

		// Return the item and the expiration time
		return item.Object, time.Unix(0, item.Expiration), true
	}

	// If expiration <= 0 (i.e. no expiration time set) then return the item
	// and a zeroed time.Time
	return item.Object, time.Time{}, true
}

func (p *cache) Flush() {
	p.items = make(map[string]*Item)
}

type keyAndValue struct {
	key   string
	value interface{}
}

// Sets an (optional) function that is called with the key and value when an
// item is evicted from the cache. (Including when it is deleted manually, but
// not when it is overwritten.) Set to nil to disable.
func (p *cache) OnEvicted(f func(string, interface{})) {
	p.mu.Lock()
	p.onEvicted = f
	p.mu.Unlock()
}

// Size
func (p *cache) Size() int {
	return len(p.items)
}

// Interval Janitor
type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (p *janitor) Run(c *cache) {
	ticker := time.NewTicker(p.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-p.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(p *cache) {
	p.janitor.stop <- true
}

func runJanitor(p *cache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	p.janitor = j
	go j.Run(p)
}

// CRUD:
func (p *cache) delete(k string) (interface{}, bool) {
	var (
		v     *Item
		found bool
	)

	if v, found = p.items[k]; found {
		delete(p.items, k)

		// Deduct usage
		p.deductMemUsage(v.Mem)

		// Delete in key manager
		p.keyManager.Delete(k)
	}

	if found && p.onEvicted != nil {
		return v.Object, true
	}

	if found {
		return v.Object, false
	}

	return nil, false
}

func (p *cache) getItem(k string) (*Item, bool) {
	item, found := p.items[k]
	if !found {
		return nil, false
	}

	// "Inlining" of Expired
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}

	return item, true
}

func (p *cache) set(k string, v interface{}, d time.Duration) error {
	var (
		e int64
	)

	// If Zero
	if d == ZeroExpiration {
		d = p.option.DefaultExpiration
	}

	// If Not Zero
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}

	// Size of Item: Value and Key
	size := p.calculateItemSize(k, v)

	// Check capacity: if seted
	if p.option.Capacity > 0 && (p.Size() >= p.option.Capacity) {
		key, err := p.keyManager.Peek()
		if err != nil {
			return err
		}

		p.delete(key)
	}

	// Check memory limit:
	if p.option.MemoryLimit > 0 && ((p.memUsage + size) >= p.option.MemoryLimit) {
		requireSpace := (p.memUsage + size) - p.option.MemoryLimit
		for requireSpace > 0 {
			key, err := p.keyManager.Peek()
			if err != nil {
				return err
			}

			if key == "" {
				return errors.New("invalid key")
			}

			item, found := p.getItem(key)

			if !found {
				continue
			}

			requireSpace = requireSpace - item.Mem
			p.delete(key)
		}

	}

	p.items[k] = &Item{
		Object:     v,
		Expiration: e,
		Mem:        size,
	}

	// Add MEM
	p.addMemUsage(size)

	// Add to key manager
	p.keyManager.Add(k)

	return nil
}

func (p *cache) get(k string) (interface{}, bool) {
	item, found := p.items[k]
	if !found {
		return nil, false
	}

	// "Inlining" of Expired
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}

	return item.Object, true
}

// MEMORY:
func (p *cache) calculateItemSize(k string, v any) int64 {
	memKey := DeepSize(k)
	memVals := DeepSize(v)
	memPointer := PtrSize

	// fmt.Printf("Key: %d, Val: %d, Pointer: %d\n", memKey, memVals, memPointer)

	return memKey + memVals + int64(memPointer)
}

func (p *cache) addMemUsage(mem int64) {
	p.memUsage = p.memUsage + mem
}

func (p *cache) deductMemUsage(mem int64) {

	left := p.memUsage - mem

	if left < 0 {
		p.memUsage = 0
		return
	}

	p.memUsage = left
}
