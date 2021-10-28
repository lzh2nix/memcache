// impl memcache.Cache with ristretto
package memcache

import (
	"errors"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/golang/groupcache/singleflight"
)

// implement in golang map
type RistrettoCache struct {
	loader                     Loader        // load func
	notUsedExpiredDataAfterDur time.Duration // do not use data after duration
	sf                         *singleflight.Group

	Ristretto *ristretto.Cache
}

type ristrettoItem struct {
	value    interface{}
	ttl      time.Duration
	updateAt time.Time
}

// load data with loader
func NewWithRistretto(loader Loader, notUsedExpiredDataAfterDur time.Duration, r *ristretto.Cache) *RistrettoCache {
	m := RistrettoCache{loader: loader, notUsedExpiredDataAfterDur: notUsedExpiredDataAfterDur, Ristretto: r, sf: &singleflight.Group{}}
	return &m
}

// set key value
func (m *RistrettoCache) Set(key string, value interface{}) {
	m.SetWithTTL(key, value, 0)
}

// set with ttl
func (m *RistrettoCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {

	item := ristrettoItem{value, ttl, time.Now()}
	if m.notUsedExpiredDataAfterDur == 0 {
		m.Ristretto.Set(key, item, 1)
	} else {
		m.Ristretto.SetWithTTL(key, item, 1, m.notUsedExpiredDataAfterDur)
	}
}

// if data is expired return the old one and lazy load with Loader
func (m *RistrettoCache) Get(key string) (value interface{}, found bool) {
	v1, ok := m.Ristretto.Get(key)
	v, ok2 := v1.(ristrettoItem)
	if !ok || !ok2 {
		// key doesn't exist in cache, need load
		return m.load(key)
	}
	// update expired data
	if v.ttl != 0 && time.Now().After(v.updateAt.Add(v.ttl)) {
		go m.update(key)
	}
	return v.value, true
}

// update update exipred data
func (m *RistrettoCache) update(key string) {
	m.load(key)
}

// load by Loader
func (m *RistrettoCache) load(key string) (value interface{}, found bool) {
	if m.loader == nil {
		return nil, false
	}
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
func (m *RistrettoCache) Del(key string) {
	m.Ristretto.Del(key)
}

// purge the cache
func (m *RistrettoCache) Purge() {
	m.Ristretto.Clear()
}
