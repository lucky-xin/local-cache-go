package lcache

import (
	"github.com/patrickmn/go-cache"
	"sync"
	"time"
)

type loader[T any] func(key string) (*ExpireVal[T], error)

type LocalCache[T any] struct {
	mutex  sync.Mutex
	cache  *cache.Cache
	loader loader[T]
}

type ExpireVal[T any] struct {
	Value  T
	Expire time.Duration
}

func NewExpireVal[T any](val T, expire time.Duration) *ExpireVal[T] {
	return &ExpireVal[T]{Value: val, Expire: expire}
}

func New[T any](defaultExpiration, cleanupInterval time.Duration, loader loader[T]) (lc *LocalCache[T]) {
	lc = &LocalCache[T]{
		cache:  cache.New(defaultExpiration, cleanupInterval),
		loader: loader,
	}
	return
}

func (lc *LocalCache[T]) Lock(f func() (error, interface{})) (token interface{}, err error) {
	//返回一个实现CatchHandler接口的对象
	lc.mutex.Lock()
	err, token = f()
	lc.mutex.Unlock()
	return
}

func (lc *LocalCache[T]) GetFromCache(key string) (*T, bool) {
	if t, b := lc.cache.Get(key); b {
		if i, ok := t.(T); ok {
			return &i, true
		}
	}
	return nil, false
}

func (lc *LocalCache[T]) Get(key string) (i *T, er error) {
	return lc.GetWithLoader(key, lc.loader)
}

func (lc *LocalCache[T]) GetWithLoader(key string, apply loader[T]) (i *T, er error) {
	if t, b := lc.cache.Get(key); b {
		if c, ok := t.(T); ok {
			i = &c
			return
		}
	}

	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if t, b := lc.cache.Get(key); b {
		if i, ok := t.(T); ok {
			t = i
		}
		return
	}
	v, er := apply(key)
	if er != nil {
		return
	}
	lc.cache.Set(key, v.Value, v.Expire)
	i = &v.Value
	return
}

func (lc *LocalCache[T]) Set(key string, v *ExpireVal[T]) {
	lc.cache.Set(key, v.Value, v.Expire)
}

func (lc *LocalCache[T]) Add(values map[string]*ExpireVal[T]) {
	for k, v := range values {
		lc.cache.Set(k, v.Value, v.Expire)
	}
}
