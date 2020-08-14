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

// Package rediscluster for session provider
//
// depend on github.com/go-redis/redis
//
// go install github.com/go-redis/redis
//
// Usage:
// import(
//   _ "github.com/goasana/asana/session/redis_cluster"
//   "github.com/goasana/asana/session"
// )
//
//	func init() {
//		globalSessions, _ = session.NewManager("redis_cluster", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"127.0.0.1:7070;127.0.0.1:7071"}``)
//		go globalSessions.GC()
//	}
//
// more docs: http://asana.me/docs/module/session.md
package rediscluster

import (
	"github.com/goasana/asana/session"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

var redispder = &Provider{}

// MaxPoolSize redis_cluster max pool size
var MaxPoolSize = 1000

// SessionStore redis_cluster session store
type SessionStore struct {
	p           *redis.ClusterClient
	sid         string
	lock        sync.RWMutex
	values      map[interface{}]interface{}
	maxLifeTime int64
}

// Set value in redis_cluster session
func (rs *SessionStore) Set(key, value interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values[key] = value
	return nil
}

// Get value in redis_cluster session
func (rs *SessionStore) Get(key interface{}) interface{} {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	if v, ok := rs.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in redis_cluster session
func (rs *SessionStore) Delete(key interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	delete(rs.values, key)
	return nil
}

// Flush clear all values in redis_cluster session
func (rs *SessionStore) Flush() error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values = make(map[interface{}]interface{})
	return nil
}

// SessionID get redis_cluster session id
func (rs *SessionStore) SessionID() string {
	return rs.sid
}

// SessionRelease save session values to redis_cluster
func (rs *SessionStore) SessionRelease(w http.ResponseWriter) {
	b, err := session.EncodeGob(rs.values)
	if err != nil {
		return
	}
	c := rs.p
	c.Set(rs.sid, string(b), time.Duration(rs.maxLifeTime)*time.Second)
}

// Provider redis_cluster session provider
type Provider struct {
	maxLifeTime int64
	savePath    string
	poolSize    int
	password    string
	dbNum       int
	poolList    *redis.ClusterClient
}

// SessionInit init redis_cluster session
// savepath like redis server addr,pool size,password,dbnum
// e.g. 127.0.0.1:6379;127.0.0.1:6380,100,test,0
func (rp *Provider) SessionInit(maxLifeTime int64, savePath string) error {
	rp.maxLifeTime = maxLifeTime
	configs := strings.Split(savePath, ",")
	if len(configs) > 0 {
		rp.savePath = configs[0]
	}
	if len(configs) > 1 {
		poolsize, err := strconv.Atoi(configs[1])
		if err != nil || poolsize < 0 {
			rp.poolSize = MaxPoolSize
		} else {
			rp.poolSize = poolsize
		}
	} else {
		rp.poolSize = MaxPoolSize
	}
	if len(configs) > 2 {
		rp.password = configs[2]
	}
	if len(configs) > 3 {
		dbnum, err := strconv.Atoi(configs[3])
		if err != nil || dbnum < 0 {
			rp.dbNum = 0
		} else {
			rp.dbNum = dbnum
		}
	} else {
		rp.dbNum = 0
	}

	rp.poolList = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    strings.Split(rp.savePath, ";"),
		Password: rp.password,
		PoolSize: rp.poolSize,
	})
	return rp.poolList.Ping().Err()
}

// SessionRead read redis_cluster session by sid
func (rp *Provider) SessionRead(sid string) (session.Store, error) {
	var kv map[interface{}]interface{}
	kvs, err := rp.poolList.Get(sid).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if len(kvs) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		if kv, err = session.DecodeGob([]byte(kvs)); err != nil {
			return nil, err
		}
	}

	rs := &SessionStore{p: rp.poolList, sid: sid, values: kv, maxLifeTime: rp.maxLifeTime}
	return rs, nil
}

// SessionExist check redis_cluster session exist by sid
func (rp *Provider) SessionExist(sid string) bool {
	c := rp.poolList
	if existed, err := c.Exists(sid).Result(); err != nil || existed == 0 {
		return false
	}
	return true
}

// SessionRegenerate generate new sid for redis_cluster session
func (rp *Provider) SessionRegenerate(oldsid, sid string) (session.Store, error) {
	c := rp.poolList
	if existed, err := c.Exists(oldsid).Result(); err != nil || existed == 0 {
		// oldsid doesn't exists, set the new sid directly
		// ignore error here, since if it return error
		// the existed value will be 0
		c.Set(sid, "", time.Duration(rp.maxLifeTime)*time.Second)
	} else {
		c.Rename(oldsid, sid)
		c.Expire(sid, time.Duration(rp.maxLifeTime)*time.Second)
	}
	return rp.SessionRead(sid)
}

// SessionDestroy delete redis session by id
func (rp *Provider) SessionDestroy(sid string) error {
	c := rp.poolList
	c.Del(sid)
	return nil
}

// SessionGC Impelment method, no used.
func (rp *Provider) SessionGC() {
}

// SessionAll return all activeSession
func (rp *Provider) SessionAll() int {
	return 0
}

func init() {
	session.Register("redis_cluster", redispder)
}
