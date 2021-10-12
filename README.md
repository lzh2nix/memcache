# memcache
a mem cache base on other populator cache, add following feacture 
 1. add lazy load(using expired data, and load it asynchronous)
 2. add singleflight for data loading
### setup backend db
```golang
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
```
### golang std map version
```golang
cache := memcache.NewWithStdMapCache(memcache.LoaderFunc(load),time.Millisecond*20)
cache.SetWithTTL("item-2", &Item{100}, time.Millisecond*10)
v, ok := cache.Get("item-2")
```
### ristretto version
```golang
ristretto, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
cache := memcache.NewWithRistretto(memcache.LoaderFunc(load), time.Millisecond*20, ristretto)
time.Sleep(time.Millisecond * 2) // only for ristretto
cache.SetWithTTL("item-2", &Item{100}, time.Millisecond*10)
v, ok := cache.Get("item-2")
```
