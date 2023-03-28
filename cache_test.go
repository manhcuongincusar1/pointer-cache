package cache

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	keymanager "github.com/manhcuongincusar1/pointer-cache/key_manager"
	"github.com/stretchr/testify/assert"
)

func TestWithKeyManager(t *testing.T) {
	c, err := New(&Option{
		MemoryLimit:       128,
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
	}, nil)

	c.WithKeyManager(keymanager.NewNoopManager())

	// Expect cacher can store only 2 key
	assert.Nil(t, err)
	val := [4]int64{}

	for i := 1; i <= 10; i++ {
		err := c.Set(fmt.Sprintf("%d", i), val, 20*time.Second)
		if i > 2 {
			assert.NotNil(t, err)
			assert.Equal(t, "invalid key", err.Error())
		} else {
			assert.Nil(t, err)
		}
	}

	assert.Equal(t, 2, c.Size())
}

func TestSetDefault(t *testing.T) {

	c, err := New(&Option{
		MemoryLimit:       1024,
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   10 * time.Millisecond,
		Capacity:          2,
	}, nil)

	assert.Nil(t, err)

	c.SetDefault("a", "Hello")

	assert.Equal(t, 1, c.keyManager.Size())

	<-time.After(25 * time.Millisecond)
	assert.Equal(t, 1, c.Size())
	_, found := c.Get("a")
	assert.True(t, found)

	// At the 51 ms later
	<-time.After(26 * time.Millisecond)
	assert.Equal(t, 0, c.Size())
	t.Log("Size", c.Size())
	_, found = c.Get("a")
	assert.False(t, found)

}

func TestFlush(t *testing.T) {
	c, err := New(&Option{
		KeyManagerType:    "",
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
		MemoryLimit:       128,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)
	val := [4]int64{}

	for i := 1; i <= 10; i++ {
		c.Set(fmt.Sprintf("%d", i), val, 20*time.Second)
	}

	c.Flush()

	assert.Equal(t, 0, c.Size())
}

func TestAdd(t *testing.T) {
	c, err := New(&Option{
		KeyManagerType:    "",
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
		MemoryLimit:       128,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)

	val := [4]int64{}

	for i := 1; i <= 10; i++ {
		c.Set(fmt.Sprintf("%d", i), val, 20*time.Second)
	}

	t.Run("SUCCESS", func(t *testing.T) {
		err := c.Add("11", "hello", 0)
		assert.Nil(t, err)
		v, f := c.Get("11")
		assert.True(t, f)
		assert.Equal(t, "hello", v.(string))
	})

	t.Run("FAIL", func(t *testing.T) {
		err := c.Add("10", "hello", 0)
		assert.NotNil(t, err)
	})
}
func TestReplace(t *testing.T) {
	c, err := New(&Option{
		KeyManagerType:    "",
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
		MemoryLimit:       128,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)

	val := [4]int64{}

	for i := 1; i <= 10; i++ {
		c.Set(fmt.Sprintf("%d", i), val, 20*time.Second)
	}

	t.Run("SUCCESS", func(t *testing.T) {
		c.Replace("10", "hello", 0)
		_val, _ := c.Get("10")
		assert.Equal(t, "hello", _val.(string))
	})

	t.Run("FAIL", func(t *testing.T) {
		c.Replace("11", "hello", 0)
		valFalse, f := c.Get("11")
		assert.Nil(t, valFalse)
		assert.False(t, f)
	})

}

func TestOverCapacity(t *testing.T) {
	c, err := New(&Option{
		KeyManagerType:    "",
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
		MemoryLimit:       100000,
		Capacity:          5,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)
	val := [4]int64{}

	for i := 1; i < 10; i++ {
		c.SetDefault(fmt.Sprintf("%d", i), val)
	}

	assert.Equal(t, 5, c.Size())
	t.Log("Size: ", c.Size())
}

func TestMemoryLimit(t *testing.T) {
	c, err := New(&Option{
		KeyManagerType:    "",
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
		MemoryLimit:       128,
		Capacity:          10,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)

	val := [4]int64{}

	for i := 1; i <= 10; i++ {
		c.SetDefault(fmt.Sprintf("%d", i), val)
		t.Log("Alloc: ", c.Alloc())
	}

	// each Item will cose 57 bytes

	assert.Equal(t, 2, c.Size())
	assert.True(t, c.option.MemoryLimit > c.memUsage)

	t.Run("Must contain the 2 last added keys (9th and 10th)", func(t *testing.T) {
		_, found := c.Get("10")
		assert.True(t, found)

		_, found = c.Get("9")
		assert.True(t, found)
	})

	t.Run("Must Not Contain 8th key", func(t *testing.T) {
		_, found := c.Get("8")
		assert.False(t, found)
	})
}

func TestNewCache(t *testing.T) {

	t.Run("SUCCESS", func(t *testing.T) {
		c, err := New(&Option{
			KeyManagerType:    "",
			DefaultExpiration: 50 * time.Millisecond,
			CleanupInterval:   1 * time.Millisecond,
			MemoryLimit:       2022,
		}, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
	})

	t.Run("FAIL_unsupported key manager", func(t *testing.T) {
		c, err := New(&Option{
			KeyManagerType:    "hello",
			DefaultExpiration: 50 * time.Millisecond,
			CleanupInterval:   1 * time.Millisecond,
			MemoryLimit:       22014,
		}, nil)

		assert.NotNil(t, err)
		assert.Nil(t, c)
	})

	t.Run("FAIL_no memory limit", func(t *testing.T) {
		c, err := New(&Option{
			KeyManagerType:  "",
			CleanupInterval: 1 * time.Millisecond,
		}, nil)

		assert.NotNil(t, err)
		assert.Nil(t, c)
	})
}

func TestDelete(t *testing.T) {
	c, err := New(&Option{
		MemoryLimit: 222222,
	}, nil)

	assert.Nil(t, err)
	assert.NotNil(t, c)

	c.Set("foo", "bar", 5000*time.Second)

	c.Delete("foo")

	x, found := c.Get("foo")
	assert.False(t, found)
	assert.Nil(t, x)
}

func TestCacheTime(t *testing.T) {
	c, err := New(&Option{
		MemoryLimit:       1024,
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   1 * time.Millisecond,
	}, nil)

	assert.Nil(t, err)

	c.Set("a", 1, ZeroExpiration)
	c.Set("b", 2, NoExpiration)
	c.Set("c", 3, 20*time.Millisecond)
	c.Set("d", 4, 70*time.Millisecond)

	assert.Equal(t, 4, c.keyManager.Size())

	<-time.After(25 * time.Millisecond)
	assert.Equal(t, 3, c.Size())
	_, found := c.Get("c")
	assert.False(t, found)

	<-time.After(30 * time.Millisecond)
	assert.Equal(t, 2, c.Size())
	_, found = c.Get("c")
	assert.False(t, found)

	<-time.After(15 * time.Millisecond)
	assert.Equal(t, 1, c.Size())
	_, found = c.Get("a")
	assert.False(t, found)
	data, found := c.Get("b")
	assert.True(t, found)
	assert.Equal(t, 2, data.(int))
}

func BenchmarkCacheGetManyConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetManyConcurrent(b, NoExpiration)
}

func benchmarkCacheGetManyConcurrent(b *testing.B, exp time.Duration) {
	// This is the same as BenchmarkCacheGetConcurrent, but its result
	// can be compared against BenchmarkShardedCacheGetManyConcurrent
	// in sharded_test.go.
	b.StopTimer()
	n := 10000
	tc, _ := New(&Option{
		KeyManagerType:    "queue",
		MemoryLimit:       1024,
		CleanupInterval:   1,
		DefaultExpiration: 1000,
	}, nil)
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		k := "foo" + strconv.Itoa(i)
		keys[i] = k
		tc.Set(k, "bar", ZeroExpiration)
	}
	each := b.N / n
	wg := new(sync.WaitGroup)
	wg.Add(n)
	for _, v := range keys {
		go func(k string) {
			for j := 0; j < each; j++ {
				tc.Get(k)
			}
			wg.Done()
		}(v)
	}
	b.StartTimer()
	wg.Wait()
}
