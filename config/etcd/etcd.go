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

// Package Kubernetes ConfigEtcd for config provider.
//
// depend on:
// "go.etcd.io/etcd/clientv3"
// "go.etcd.io/etcd/mvcc/mvccpb"
//
// Usage:
//  import(
//    _ "github.com/goasana/framework/config/etcd"
//      "github.com/goasana/framework/config"
//  )
//
//  cnf, err := NewConfig(config.EtcdProvider, "myConfAppName")
//
//More docs http://asana.me/docs/module/md
package etcd

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/goasana/framework/config"
	"github.com/goasana/framework/config/base"
	"github.com/goasana/framework/encoder"

	cetcd "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

const DefaultPrefix = "/asana/config/"

type addressKey struct{}

// ConfigConsul is a json config parser and implements ConfigConsul interface.
type ConfigConsul struct {
	prefix      string
	stripPrefix string
	client      *cetcd.Client
	cerr        error
	opts        *config.Option
}

func (e *ConfigConsul) SetOption(option config.Option) {
	e.opts = &option

	prefix, sp, client, err := getClient(option)

	e.client = client
	e.stripPrefix = sp
	e.prefix = prefix
	e.cerr = err
}

// Parse returns a ConfigConsulContainer with parsed json config map.
func (e *ConfigConsul) Parse() (config.Configer, error) {
	if e.cerr != nil {
		return nil, e.cerr
	}

	rsp, err := e.client.Get(context.Background(), e.prefix, cetcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	if rsp == nil || len(rsp.Kvs) == 0 {
		return nil, fmt.Errorf("source not found: %s", e.prefix)
	}

	var kvs []*mvccpb.KeyValue
	for _, v := range rsp.Kvs {
		kvs = append(kvs, (*mvccpb.KeyValue)(v))
	}

	data := makeMap(e.opts.Encoder, kvs, e.stripPrefix)

	b, err := e.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	return e.ParseData(b)
}

// ParseData json data
func (e *ConfigConsul) ParseData(data []byte) (config.Configer, error) {
	x := &ConfigConsulContainer{ConfigBaseContainer: base.ConfigBaseContainer{
		Data:          make(map[string]interface{}),
		SeparatorKeys: e.opts.SeparatorKeys,
	}}

	cnf := map[string]interface{}{}
	_ = e.opts.Encoder.Decode(data, &cnf)

	x.Data = config.ExpandValueEnvForMap(cnf)

	return x, nil
}

// ConfigConsulContainer A ConfigConsul represents the json configuration.
type ConfigConsulContainer struct {
	base.ConfigBaseContainer
}

func NewConfigConsul(option config.Option) *ConfigConsul {
	prefix, sp, client, err := getClient(option)

	return &ConfigConsul{
		prefix:      prefix,
		stripPrefix: sp,
		opts:        &option,
		client:      client,
		cerr:        err,
	}
}

func init() {
	config.Register(config.EtcdProvider, NewConfigConsul(config.Option{
		ConfigName:    DefaultPrefix,
		SeparatorKeys: "::",
		Context:       context.Background(),
	}))
}

func makeMap(e encoder.Encoder, kv []*mvccpb.KeyValue, stripPrefix string) map[string]interface{} {
	data := make(map[string]interface{})

	for _, v := range kv {
		data = update(e, data, v, "put", stripPrefix)
	}

	return data
}

func update(e encoder.Encoder, data map[string]interface{}, v *mvccpb.KeyValue, action, stripPrefix string) map[string]interface{} {
	// remove prefix if non empty, and ensure leading / is removed as well
	vkey := strings.TrimPrefix(strings.TrimPrefix(string(v.Key), stripPrefix), "/")
	// split on prefix
	haveSplit := strings.Contains(vkey, "/")
	keys := strings.Split(vkey, "/")

	var vals interface{}
	_ = e.Decode(v.Value, &vals)

	if !haveSplit && len(keys) == 1 {
		switch action {
		case "delete":
			data = make(map[string]interface{})
		default:
			v, ok := vals.(map[string]interface{})
			if ok {
				data = v
			}
		}
		return data
	}

	// set data for first iteration
	kvals := data
	// iterate the keys and make maps
	for i, k := range keys {
		kval, ok := kvals[k].(map[string]interface{})
		if !ok {
			// create next map
			kval = make(map[string]interface{})
			// set it
			kvals[k] = kval
		}

		// last key: write vals
		if l := len(keys) - 1; i == l {
			switch action {
			case "delete":
				delete(kvals, k)
			default:
				kvals[k] = vals
			}
			break
		}

		// set kvals for next iterator
		kvals = kval
	}

	return data
}

func getClient(option config.Option) (string, string, *cetcd.Client, error) {
	endpoints := []string{"localhost:2379"}

	// check if there are any addrs
	a, ok := option.Context.Value(addressKey{}).(string)
	if ok {
		addr, port, err := net.SplitHostPort(a)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "2379"
			addr = a
			endpoints = []string{fmt.Sprintf("%s:%s", addr, port)}
		} else if err == nil {
			endpoints = []string{fmt.Sprintf("%s:%s", addr, port)}
		}
	}

	c, e := cetcd.New(cetcd.Config{
		Endpoints: endpoints,
	})
	// use default config
	return option.ConfigName, "", c, e
}
