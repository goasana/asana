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

// Package Kubernetes ConfigFile for config provider.
//
// depend on:
// "k8s.io/apimachinery/pkg/apis/meta/v1"
// "k8s.io/client-go/kubernetes"
// "k8s.io/client-go/rest"
//
// Usage:
//  import(
//    _ "github.com/goasana/framework/config/file"
//      "github.com/goasana/framework/config"
//  )
//
//  cnf, err := NewConfig(config.FileProvider, "/conf/conf.json")
//
//More docs http://asana.me/docs/module/md
package file

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	"github.com/goasana/framework/config"
	"github.com/goasana/framework/config/base"
	"github.com/goasana/framework/encoder"
)

// ConfigFile is a json config parser and implements ConfigFile interface.
type ConfigFile struct {
	opts *config.Option
}

func (f *ConfigFile) SetOption(option config.Option) {
	f.opts = &option
}

// Parse returns a ConfigFileContainer with parsed config map.
func (f *ConfigFile) Parse() (config.Configer, error) {
	fh, err := os.Open(f.opts.ConfigName)

	if err != nil {
		return nil, err
	}

	defer fh.Close()

	b, err := ioutil.ReadAll(fh)
	if err != nil {
		return nil, err
	}

	f.opts.Encoder = encoder.GetEncoder(format(f.opts.ConfigName, f.opts.Encoder))

	return f.ParseData(b)
}

func format(p string, e encoder.Encoder) string {
	parts := strings.Split(p, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return e.String()
}

// ParseData json data
func (f *ConfigFile) ParseData(data []byte) (config.Configer, error) {
	x := &ConfigFileContainer{ConfigBaseContainer: base.ConfigBaseContainer{
		Data:          make(map[string]interface{}),
		SeparatorKeys: f.opts.SeparatorKeys,
	}}

	cnf := map[string]interface{}{}
	_ = f.opts.Encoder.Decode(data, &cnf)

	x.ConfigBaseContainer.Data = config.ExpandValueEnvForMap(cnf)

	return x, nil
}

// ConfigFileContainer A ConfigFile represents the json configuration.
type ConfigFileContainer struct {
	base.ConfigBaseContainer
}

func NewConfigFile(option config.Option) *ConfigFile {
	return &ConfigFile{
		opts: &option,
	}
}

func init() {
	config.Register(config.FileProvider, NewConfigFile(config.Option{
		SeparatorKeys: "::",
		Context:       context.Background(),
	}))
}
