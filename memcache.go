package memcache

import (
	"time"
)

// simple in memory cache interface
type Cache interface {
	Set(key string, value interface{})
	SetWithTTL(key string, value interface{}, ttl time.Duration)
	// if data is expired return the old one and lazy load with Loader
	Get(key string) (value interface{}, found bool)
	Del(key string)
	Purge()
}

// A Loader loads data for a key.
type Loader interface {
	Load(key string) (value interface{}, ttl time.Duration, err error)
}

// A LoaderFunc implements Loader with a function.
type LoaderFunc func(key string) (value interface{}, ttl time.Duration, err error)

func (f LoaderFunc) Load(key string) (value interface{}, ttl time.Duration, err error) {
	return f(key)
}
