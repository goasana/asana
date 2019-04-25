package cache

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/bluele/gcache"
)

// Cache gCache adapter
type gCache struct {
	cache gcache.Cache
}

//NewSsdbCache create new gCache adapter.
func NewgCache() Cache {
	return &gCache{}
}

// Get get value from memcache.
func (rc *gCache) Get(key string) interface{} {
	value, err := rc.cache.Get(key)
	if err == nil {
		return value
	}
	return nil
}

// GetMulti get value from memcache.
func (rc *gCache) GetMulti(keys []string) []interface{} {
	values := make([]interface{}, 0)
	for _, key := range keys {
		values = append(values, rc.Get(key))
	}
	return values
}

// DelMulti get value from memcache.
func (rc *gCache) DelMulti(keys []string) error {
	for _, key := range keys {
		rc.Delete(key)
	}
	return nil
}

// Put put value to memcache. only support string.
func (rc *gCache) Put(key string, value interface{}, timeout time.Duration) error {
	if timeout == 0 {
		return rc.cache.Set(key, value)
	}
	return rc.cache.SetWithExpire(key, value, timeout)
}

// Delete delete value in memcache.
func (rc *gCache) Delete(key string) error {
	rc.cache.Remove(key)
	return nil
}

// Incr increase counter.
func (rc *gCache) Incr(key string) error {
	val, err := rc.cache.Get(key)

	if err != nil {
		return err
	}

	switch v := val.(type) {
	case float32:
		err = rc.cache.Set(key, float32(v)+1)
	case float64:
		err = rc.cache.Set(key, float64(v)+1)
	case int8:
		err = rc.cache.Set(key, int8(v)+1)
	case int16:
		err = rc.cache.Set(key, int16(v)+1)
	case int32:
		err = rc.cache.Set(key, int32(v)+1)
	case int64:
		err = rc.cache.Set(key, int64(v)+1)
	case uint:
		err = rc.cache.Set(key, uint(v)+1)
	case uint8:
		err = rc.cache.Set(key, uint8(v)+1)
	case uint16:
		err = rc.cache.Set(key, uint16(v)+1)
	case uint32:
		err = rc.cache.Set(key, uint32(v)+1)
	case uint64:
		err = rc.cache.Set(key, uint64(v)+1)
	case int:
		err = rc.cache.Set(key, int(v)+1)
	default:
		return errors.New("is not numeric")
	}
	return err
}

// Decr decrease counter.
func (rc *gCache) Decr(key string) error {
	val, err := rc.cache.Get(key)

	if err != nil {
		return err
	}

	switch v := val.(type) {
	case float32:
		err = rc.cache.Set(key, float32(v)-1)
	case float64:
		err = rc.cache.Set(key, float64(v)-1)
	case int8:
		err = rc.cache.Set(key, int8(v)-1)
	case int16:
		err = rc.cache.Set(key, int16(v)-1)
	case int32:
		err = rc.cache.Set(key, int32(v)-1)
	case int64:
		err = rc.cache.Set(key, int64(v)-1)
	case uint:
		err = rc.cache.Set(key, uint(v)-1)
	case uint8:
		err = rc.cache.Set(key, uint8(v)-1)
	case uint16:
		err = rc.cache.Set(key, uint16(v)-1)
	case uint32:
		err = rc.cache.Set(key, uint32(v)-1)
	case uint64:
		err = rc.cache.Set(key, uint64(v)-1)
	case int:
		err = rc.cache.Set(key, int(v)-1)
	default:
		return errors.New("is not numeric")
	}
	return err
}

// IsExist check value exists in memcache.
func (rc *gCache) IsExist(key string) bool {
	return rc.cache.Has(key)
}

// ClearAll clear all cached in memcache.
func (rc *gCache) ClearAll() error {
	rc.cache.Purge()
	return nil
}

// StartAndGC start gCache adapter.
// if connecting error, return.
func (rc *gCache) StartAndGC(config string) error {
	var cf map[string]interface{}
	json.Unmarshal([]byte(config), &cf)
	if _, ok := cf["size"]; !ok {
		cf["size"] = 30
	}

	switch GetString(cf["type"]) {
	case "arc":
		rc.cache = gcache.New(GetInt(cf["size"])).ARC().Build()
	case "lru":
		rc.cache = gcache.New(GetInt(cf["size"])).LRU().Build()
	case "lfu":
		rc.cache = gcache.New(GetInt(cf["size"])).LFU().Build()
	default:
		rc.cache = gcache.New(GetInt(cf["size"])).Build()
	}

	return nil
}

func init() {
	Register(GCacheProvider, NewgCache)
}
