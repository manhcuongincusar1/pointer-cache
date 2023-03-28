package keymanager

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueUnlimit(t *testing.T) {
	km := NewQueue(0)
	for i := 0; i < 1000000; i++ {
		added := km.Add(fmt.Sprintf("Key%d", i))
		assert.True(t, added)
	}

	assert.Equal(t, 1000000, km.Size())
}

func TestPeek(t *testing.T) {
	km := NewQueue(3)

	t.Run("Fail When Empty", func(t *testing.T) {
		key, err := km.Peek()

		assert.NotNil(t, err)
		assert.Equal(t, "", key)
	})

	for i := 0; i < 2; i++ {
		added := km.Add(fmt.Sprintf("Key%d", i))
		assert.True(t, added)
	}

	t.Run("Success Found", func(t *testing.T) {
		key, err := km.Peek()
		assert.Nil(t, err)
		assert.Equal(t, "Key0", key)
	})
}

func TestAddItem(t *testing.T) {
	km := NewQueue(3)

	for i := 0; i < 10; i++ {
		added := km.Add(fmt.Sprintf("key%d", i))

		if i >= 3 {
			t.Run("Should Fail", func(t *testing.T) {
				assert.True(t, !added)
			})
		}

		if i < 3 {
			t.Run("Should True", func(t *testing.T) {
				assert.True(t, added)
			})
		}

	}

	t.Run("Size Less than 3", func(t *testing.T) {
		assert.Equal(t, 3, km.Size())
	})

}

func TestQueue_Clear(t *testing.T) {
	q := queue{
		size:  10,
		array: make([]string, 0, 10),
	}
	assert.Equal(t, q.Size(), 0)
	assert.Equal(t, q.IsEmpty(), true)

	q.Enqueue("key1")
	assert.Equal(t, q.IsEmpty(), false)
	assert.Equal(t, q.Size(), 1)

	q.Clear()
	assert.Equal(t, q.IsEmpty(), true)
	assert.Equal(t, q.Size(), 0)
}

func TestQueue_GetValues(t *testing.T) {
	q := queue{
		size:  10,
		array: make([]string, 0, 10),
	}

	q.Enqueue("key1", "key2", "key3")
	assert.True(t, reflect.DeepEqual(q.GetValues(), []string{"key1", "key2", "key3"}))
}

func TestQueue_Dequeue(t *testing.T) {
	q := queue{
		size:  10,
		array: make([]string, 0, 10),
	}

	q.Enqueue("key1", "key2")

	value, err := q.Dequeue()
	assert.Nil(t, err)
	assert.Equal(t, q.IsEmpty(), false)
	assert.Equal(t, q.Size(), 1)
	assert.Equal(t, value, "key1")

	value, err = q.Dequeue()
	assert.Nil(t, err)
	assert.Equal(t, q.IsEmpty(), true)
	assert.Equal(t, q.Size(), 0)
	assert.Equal(t, value, "key2")

	_, err = q.Dequeue()
	assert.NotNil(t, err)
}

func TestDelete(t *testing.T) {

	q := queue{
		size:  5,
		array: []string{"1", "2", "3", "4", "5"},
	}

	// Case delete last ("5")
	q.Delete("5")
	assert.True(t, reflect.DeepEqual(q.array, []string{"1", "2", "3", "4"}))

	// Case delete first
	q.Delete("1")
	assert.True(t, reflect.DeepEqual(q.array, []string{"2", "3", "4"}))

	// Case delete in middle
	q.Delete("3")
	assert.True(t, reflect.DeepEqual(q.array, []string{"2", "4"}))
}
