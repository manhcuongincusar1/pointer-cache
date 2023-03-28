package keymanager

import (
	"errors"
	"sync"
)

func NewQueue(size uint32, values ...string) KeyManager {
	var arr []string

	if size != 0 {
		arr = make([]string, 0, size)
	}

	return &queue{
		array: arr,
		size:  size,
	}
}

var (
	errEmptyQueue = errors.New("queue is empty")
)

// Queue Queue structure
type queue struct {
	size  uint32
	array []string
	mu    sync.RWMutex
}

// Implement KeyManager
// Add new key
func (p *queue) Add(key string) (added bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.size != 0 && (p.Size() >= int(p.size)) {
		return false
	}
	p.Enqueue(key)

	return true
}

// remove from arr
func remove[T comparable](l []T, item T) []T {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

// Delete when cache remove key
func (p *queue) Delete(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.array = remove(p.array, key)
}

// Remove the oldest key
func (p *queue) Shift() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key, err := p.Dequeue()
	return key, err
}

// Size off current
func (p *queue) Size() int {
	return len(p.array)
}

// Enqueue add to the Queue
func (p *queue) Enqueue(values ...string) {
	p.array = append(p.array, values...)
}

// IsEmpty checks if the Queue is empty
func (p *queue) IsEmpty() bool {
	return p.Size() == 0
}

// Clear clears Queue
func (p *queue) Clear() {
	p.array = nil
}

// Dequeue remove from the Queue
func (p *queue) Dequeue() (res string, err error) {
	if p.IsEmpty() {
		return res, errEmptyQueue
	}

	res = p.array[0]
	p.array = p.array[1:]
	return res, nil
}

// Peek returns front of the Queue
func (p *queue) Peek() (res string, err error) {
	if p.IsEmpty() {
		return res, errEmptyQueue
	}

	res = p.array[0]
	return res, nil
}

// GetValues returns values
func (p *queue) GetValues() []string {
	values := make([]string, 0, p.Size())
	for _, value := range p.array {
		values = append(values, value)
	}
	return values
}
