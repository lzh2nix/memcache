// impl memcache.Cache with std map
package memcache

import (
	"errors"
	"sync"
	"time"

	"github.com/golang/groupcache/singleflight"
)

// implement in golang map
type StdMapCache struct {
	loader                     Loader        // load func
	notUsedExpiredDataAfterDur time.Duration // do not use data after duration
	sf                         *singleflight.Group

	lock  sync.RWMutex
	items map[string]item
}

type item struct {
	value    interface{}
	ttl      time.Duration
	updateAt time.Time
}

// load data with loader
func NewWithStdMapCache(loader Loader, notUsedExpiredDataAfterDur time.Duration) *StdMapCache {
	m := StdMapCache{loader: loader, notUsedExpiredDataAfterDur: notUsedExpiredDataAfterDur, items: map[string]item{}, sf: &singleflight.Group{}}
	return &m
}

// set key value
func (m *StdMapCache) Set(key string, value interface{}) {
	m.SetWithTTL(key, value, 0)
}

// set with ttl
func (m *StdMapCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items[key] = item{value, ttl, time.Now()}

}

// if data is expired return the old one and lazy load with Loader
func (m *StdMapCache) Get(key string) (value interface{}, found bool) {
	m.lock.RLock()
	v, ok := m.items[key]
	m.lock.RUnlock()
	if !ok {
		// key doesn't exist in cache, need load
		return m.load(key)
	}

	if v.ttl != 0 && time.Now().After(v.updateAt.Add(m.notUsedExpiredDataAfterDur)) {
		// it's too old, need load
		return m.load(key)
	}

	// update expired data
	if v.ttl != 0 && time.Now().After(v.updateAt.Add(v.ttl)) {
		go m.update(key)
	}
	return v.value, true
}

// update update exipred data
func (m *StdMapCache) update(key string) {
	m.load(key)
}

// load by Loader
func (m *StdMapCache) load(key string) (value interface{}, found bool) {
	v, err := m.sf.Do(key, func() (interface{}, error) {
		value, ttl, err := m.loader.Load(key)
		if err == nil && value != nil {
			m.SetWithTTL(key, value, ttl)
			return value, nil
		}
		return nil, errors.New("load failed")

	})
	if err != nil {
		return nil, false
	}
	return v, true
}

// delete from cache
func (m *StdMapCache) Del(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.items, key)
}

// purge the cache
func (m *StdMapCache) Purge() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items = map[string]item{}
}
