// Copyright 2019 asana Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache provide a Cache interface and some implement engine
// Usage:
//
// import(
//   "github.com/goasana/asana/cache"
// )
//
// bm, err := cache.NewCache(cache.MemoryProvider, `{"interval":60}`)
//
// Use it like this:
//
//	bm.Put("asana", 1, 10 * time.Second)
//	bm.Get("asana")
//	bm.IsExist("asana")
//	bm.Delete("asana")
//
//  more docs http://asana.me/docs/module/cache.md
package cache

import (
	"fmt"
	"time"
)

// Cache interface contains all behaviors for cache adapter.
// usage:
//	cache.Register("file",cache.NewFileCache) // this operation is run in init method of file.go.
//	c,err := cache.NewCache(cache.FileProvider,"{....}")
//	c.Put("key",value, 3600 * time.Second)
//	v := c.Get("key")
//
//	c.Incr("counter")  // now is 1
//	c.Incr("counter")  // now is 2
//	count := c.Get("counter").(int)
type Cache interface {
	// get cached value by key.
	Get(key string) interface{}
	// GetMulti is a batch version of Get.
	GetMulti(keys []string) []interface{}
	// set cached value with key and expire time.
	Put(key string, val interface{}, timeout time.Duration) error
	// delete cached value by key.
	Delete(key string) error
	// increase cached int value by key, as a counter.
	Incr(key string) error
	// decrease cached int value by key, as a counter.
	Decr(key string) error
	// check if cached value exists or not.
	IsExist(key string) bool
	// clear all cache.
	ClearAll() error
	// start gc routine based on config string settings.
	StartAndGC(config string) error
}

// Provider type
type Provider string

// Provider Avails
const (
	RedisProvider     Provider = "redis"
	MemCachedProvider Provider = "memcache"
	SSDBProvider      Provider = "ssdb"
	GCacheProvider    Provider = "gCache"
	MemoryProvider    Provider = "memory"
	FileProvider      Provider = "file"
)

// Instance is a function create a new Cache Instance
type Instance func() Cache

var adapters = make(map[Provider]Instance)

// Register makes a cache adapter available by the adapter name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(provider Provider, adapter Instance) {
	if adapter == nil {
		panic("cache: Register adapter is nil")
	}
	if _, ok := adapters[provider]; ok {
		panic("cache: Register called twice for adapter " + provider)
	}
	adapters[provider] = adapter
}

// NewCache Create a new cache driver by adapter name and config string.
// config need to be correct JSON as string: {"interval":360}.
// it will start gc automatically.
func NewCache(provider Provider, config string) (adapter Cache, err error) {
	instanceFunc, ok := adapters[provider]
	if !ok {
		err = fmt.Errorf("cache: unknown adapter name %q (forgot to import?)", provider)
		return
	}
	adapter = instanceFunc()
	err = adapter.StartAndGC(config)
	if err != nil {
		adapter = nil
	}
	return
}
