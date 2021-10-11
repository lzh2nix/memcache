package memcache_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/lzh2nix/memcache"
	"github.com/stretchr/testify/require"
)

type Item struct {
	I int
}

var m map[string]*Item

func init() {
	m = map[string]*Item{}
	for i := 0; i < 1000; i++ {
		m[fmt.Sprintf("item-%d", i)] = &Item{i}
	}
}

func load(key string) (value interface{}, ttl time.Duration, err error) {
	v, ok := m[key]
	if !ok {
		return nil, 0, errors.New("not found")
	}
	return v, time.Millisecond * 10, nil
}

func test(t *testing.T, c memcache.Cache) {
	ast := require.New(t)
	{
		// test get from backend if data miss
		v, ok := c.Get("item-1")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 1)
	}
	{
		// test delete
		c.Del("item-1")
		v, ok := c.Get("item-1")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 1)
	}
	{
		// return old data if expired and lazy load
		c.SetWithTTL("item-2", &Item{100}, time.Millisecond*10)
		time.Sleep(time.Millisecond * 2)
		v, ok := c.Get("item-2")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 100)
		time.Sleep(time.Millisecond * 10)
		// return old data
		v, ok = c.Get("item-2")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 100)
		time.Sleep(time.Millisecond * 10)
		// get lazy loaded data
		v, ok = c.Get("item-2")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 2)
	}
	{
		_, ok := c.Get("item-1111111111")
		ast.False(ok)
	}
	{
		// too old need load
		c.SetWithTTL("item-3", &Item{100}, time.Millisecond*10)
		time.Sleep(time.Millisecond * 5)
		v, ok := c.Get("item-3")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 100)
		time.Sleep(time.Millisecond * 20)
		v, ok = c.Get("item-3")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 3)
	}
	{
		c.Set("item-4", &Item{200})
		time.Sleep(time.Millisecond * 2)

		v, ok := c.Get("item-4")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 200)
		time.Sleep(time.Millisecond * 20)
		v, ok = c.Get("item-4")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 200)
	}
	{
		c.Set("item-100000", &Item{10000})
		time.Sleep(time.Millisecond * 2)
		v, ok := c.Get("item-100000")
		ast.True(ok)
		ast.Equal(v.(*Item).I, 10000)
		c.Purge()
		v, ok = c.Get("item-100000")
		ast.False(ok)
	}
}

// TestCache ...
func TestStdMapVersion(t *testing.T) {
	c := memcache.NewWithStdMapCache(memcache.LoaderFunc(load), time.Millisecond*20)
	test(t, c)
}

func TestRistrettoVersion(t *testing.T) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
	c := memcache.NewWithRistretto(memcache.LoaderFunc(load), time.Millisecond*20, cache)
	test(t, c)

}
