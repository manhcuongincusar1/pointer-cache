package cache

import (
	"errors"
	"runtime"

	keymanager "github.com/manhcuongincusar1/pointer-cache/key_manager"
)

func New(option *Option, initData map[string]*Item) (*Cache, error) {
	var _cache *cache
	if option.MemoryLimit == 0 {
		return nil, errors.New("memory limit is required")
	}

	_cache, err := newCacheWithJanitor(option, initData)
	if err != nil {
		return nil, err
	}

	// keymanager
	keyManager, err := keymanager.NewKeyManager(option.KeyManagerType, 0)
	if err != nil {
		return nil, err
	}
	_cache.keyManager = keyManager

	return &Cache{
		_cache,
	}, nil
}

type Cache struct {
	*cache
}

func newCache(option *Option, m map[string]*Item) *cache {
	if m == nil {
		m = make(map[string]*Item)
	}

	c := &cache{
		option: option,
		items:  m,
	}

	return c
}

func newCacheWithJanitor(option *Option, m map[string]*Item) (*cache, error) {
	c := newCache(option, m)
	if option.CleanupInterval > 0 {
		runJanitor(c, option.CleanupInterval)
		runtime.SetFinalizer(c, stopJanitor)
	}

	return c, nil
}
