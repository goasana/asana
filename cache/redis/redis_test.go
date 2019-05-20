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

package redis

import (
	"strconv"
	"testing"
	"time"

	"github.com/goasana/asana/cache"
	"github.com/gomodule/redigo/redis"
)

func TestRedisCache(t *testing.T) {
	bm, err := cache.NewCache(cache.RedisProvider, `{"conn": "127.0.0.1:6379"}`)
	if err != nil {
		t.Error("init err")
	}
	timeoutDuration := 10 * time.Second
	if err = bm.Put("asana", 1, timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if !bm.IsExist("asana") {
		t.Error("check err")
	}

	time.Sleep(11 * time.Second)

	if bm.IsExist("asana") {
		t.Error("check err")
	}
	if err = bm.Put("asana", 1, 0); err != nil {
		t.Error("set Error", err)
	}

	if v, _ := strconv.Atoi(bm.Get("asana").(string)); v != 1 {
		t.Error("get err")
	}

	if err = bm.Incr("asana"); err != nil {
		t.Error("Incr Error", err)
	}

	if v, _ := strconv.Atoi(bm.Get("asana").(string)); v != 2 {
		t.Error("get err")
	}

	if err = bm.Decr("asana"); err != nil {
		t.Error("Decr Error", err)
	}

	if v, _ := strconv.Atoi(bm.Get("asana").(string)); v != 1 {
		t.Error("get err")
	}
	_ = bm.Delete("asana")
	if bm.IsExist("asana") {
		t.Error("delete err")
	}

	//test string
	if err = bm.Put("asana", "author", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if !bm.IsExist("asana") {
		t.Error("check err")
	}

	if v, _ := redis.String(bm.Get("asana"), err); v != "author" {
		t.Error("get err")
	}

	//test GetMulti
	if err = bm.Put("asana1", "author1", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if !bm.IsExist("asana1") {
		t.Error("check err")
	}

	vv := bm.GetMulti([]string{"asana", "asana1"})
	if len(vv) != 2 {
		t.Error("GetMulti ERROR")
	}
	if v := vv[0]; v != "author" {
		t.Error("GetMulti ERROR")
	}
	if v := vv[1]; v != "author1" {
		t.Error("GetMulti ERROR")
	}

	// test clear all
	if err = bm.ClearAll(); err != nil {
		t.Error("clear all err")
	}
}
