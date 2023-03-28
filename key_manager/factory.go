package keymanager

import "errors"

func NewKeyManager(holder string, size uint32) (KeyManager, error) {

	if size == 0 {
		size = 0
	}

	if holder == "" || holder == "queue" {
		return NewQueue(size), nil
	}

	return nil, errors.New("unsupported key manager")
}
