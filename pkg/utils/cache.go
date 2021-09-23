package utils

import (
	"sync"
	"time"
)

type Cache struct {
	data   map[string]interface{}
	expiry map[string]time.Time
	lock   sync.RWMutex

	// Utility lock for external handling of cache
	sharedLock sync.Mutex
}

func NewCache() *Cache {
	cache := Cache{}
	cache.data = make(map[string]interface{})
	cache.expiry = make(map[string]time.Time)
	cache.lock = sync.RWMutex{}
	cache.sharedLock = sync.Mutex{}
	return &cache
}

func (cache *Cache) Get(key string) interface{} {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	if expiry, found := cache.expiry[key]; found {
		if expiry.Before(time.Now()) {
			delete(cache.data, key)
			delete(cache.expiry, key)
		}
	}

	cachedValue, exists := cache.data[key]
	if !exists {
		return nil
	}
	return cachedValue
}

func (cache *Cache) Keys() []string {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	now := time.Now()
	for key, value := range cache.expiry {
		if value.Before(now) {
			delete(cache.data, key)
			delete(cache.expiry, key)
		}
	}

	var keys []string

	for key := range cache.data {
		keys = append(keys, key)
	}

	return keys
}

func (cache *Cache) Set(key string, value interface{}) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.data[key] = value
}

func (cache *Cache) SetWithExpiry(key string, value interface{}, t time.Time) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.data[key] = value
	cache.expiry[key] = t
}

func (cache *Cache) SetWithDuration(key string, value interface{}, d time.Duration) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.data[key] = value
	cache.expiry[key] = time.Now().Add(d)
}

func (cache *Cache) Delete(key string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	delete(cache.data, key)
	delete(cache.expiry, key)
}

func (cache *Cache) DeleteAll() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.data = make(map[string]interface{})
	cache.expiry = make(map[string]time.Time)
}

func (cache *Cache) Lock() {
	cache.sharedLock.Lock()
}
func (cache *Cache) Unlock() {
	cache.sharedLock.Unlock()
}
