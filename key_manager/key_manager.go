package keymanager

type KeyManager interface {
	Add(key string) bool
	Size() int
	Delete(key string)     // Delete the key
	Peek() (string, error) // Take the first option
}
