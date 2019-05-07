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

// Package goredis for cache provider
//
// depend on github.com/go-redis/redis
//
// go install github.com/go-redis/redis
//
// Usage:
// import(
//   _ "github.com/goasana/asana/cache/redis"
//   "github.com/goasana/asana/cache"
// )
//
//  bm, err := cache.NewCache("redis", `{"conn":"127.0.0.1:11211"}`)
//
//  more docs http://asana.me/docs/module/cache.md
package goredis

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/goasana/config/encoder/json"
	"github.com/goasana/asana/cache"
)

var (
	// DefaultKey the collection name of redis for cache adapter.
	DefaultKey = "asanaCacheRedis"
)

// Cache is Redis cache adapter.
type Cache struct {
	p        *redis.Client // redis connection pool
	conninfo string
	dbNum    int
	key      string
	password string
	maxIdle  int
}

// NewGoRedisCache create new redis cache with default collection name.
func NewGoRedisCache() cache.Cache {
	return &Cache{key: DefaultKey}
}

// Get cache from redis.
func (rc *Cache) Get(key string) interface{} {
	v, err := rc.p.Get(key).Result()
	if err == nil {
		return v
	}
	return nil
}

// GetMulti get cache from redis.
func (rc *Cache) GetMulti(keys []string) []interface{} {
	c := rc.p.MGet(keys...)
	values, err := c.Result()
	if err != nil {
		return nil
	}
	return values
}

// Put put cache to redis.
func (rc *Cache) Put(key string, val interface{}, timeout time.Duration) error {
	_, err := rc.p.Set(key, val, timeout).Result()
	return err
}

// Delete delete cache in redis.
func (rc *Cache) Delete(key string) (err error) {
	c := rc.p.Del(key)

	_, err = c.Result()

	return
}

// IsExist check cache's existence in redis.
func (rc *Cache) IsExist(key string) bool {
	v, err := rc.p.Exists(key).Result()

	if err != nil {
		return false
	}

	return v > 0
}

// Incr increase counter in redis.
func (rc *Cache) Incr(key string) error {
	_, err := rc.p.Incr(key).Result()
	return err
}

// Decr decrease counter in redis.
func (rc *Cache) Decr(key string) error {
	_, err := rc.p.Decr(key).Result()
	return err
}

// ClearAll clean all cache in redis. delete this redis collection.
func (rc *Cache) ClearAll() error {
	c := rc.p.FlushDB()
	return c.Err()
}

// StartAndGC start redis cache adapter.
// config is like {"key":"collection key","conn":"connection info","dbNum":"0"}
// the cache item in redis are stored forever,
// so no gc operation.
func (rc *Cache) StartAndGC(config string) error {
	var cf map[string]string
	_ = json.Decode([]byte(config), &cf)

	if _, ok := cf["key"]; !ok {
		cf["key"] = DefaultKey
	}
	if _, ok := cf["conn"]; !ok {
		return errors.New("config has no conn key")
	}

	// Format redis://<password>@<host>:<port>
	cf["conn"] = strings.Replace(cf["conn"], "redis://", "", 1)
	if i := strings.Index(cf["conn"], "@"); i > -1 {
		cf["password"] = cf["conn"][0:i]
		cf["conn"] = cf["conn"][i+1:]
	}

	if _, ok := cf["dbNum"]; !ok {
		cf["dbNum"] = "0"
	}
	if _, ok := cf["password"]; !ok {
		cf["password"] = ""
	}
	if _, ok := cf["maxIdle"]; !ok {
		cf["maxIdle"] = "3"
	}
	rc.key = cf["key"]
	rc.conninfo = cf["conn"]
	rc.dbNum, _ = strconv.Atoi(cf["dbNum"])
	rc.password = cf["password"]
	rc.maxIdle, _ = strconv.Atoi(cf["maxIdle"])

	rc.connectInit()

	c := rc.p.Ping()

	return c.Err()
}

// connect to redis.
func (rc *Cache) connectInit() {
	// initialize a new pool
	rc.p = redis.NewClient(&redis.Options{
		Addr:     rc.conninfo,
		DB:       rc.dbNum,
		Password: rc.password,
	},
	)
}

func init() {
	cache.Register(cache.GoRedisProvider, NewGoRedisCache)
}
