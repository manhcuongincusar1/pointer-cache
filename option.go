package cache

import "time"

type Option struct {
	KeyManagerType    string
	Capacity          int
	MemoryLimit       int64
	CleanupInterval   time.Duration
	DefaultExpiration time.Duration
}
