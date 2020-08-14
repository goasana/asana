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

package asana

import (
	"reflect"
	"testing"

	"github.com/goasana/config/encoder/json"
	"github.com/goasana/config"
	"github.com/goasana/config/source/memory"
)

func TestDefaults(t *testing.T) {
	if BConfig.WebConfig.FlashName != "ASANA_FLASH" {
		t.Errorf("FlashName was not set to default.")
	}

	if BConfig.WebConfig.FlashSeparator != "ASANAFLASH" {
		t.Errorf("FlashName was not set to default.")
	}
}

func TestAssignConfig_01(t *testing.T) {
	_BConfig := &Config{}
	_BConfig.AppName = "asana_test"
	jcf := config.NewConfig()

	_ = jcf.Load(memory.NewSource(memory.WithJson([]byte(`{"AppName":"asana_json"}`))))

	assignSingleConfig(_BConfig, jcf)
	if _BConfig.AppName != "asana_json" {
		t.Log(_BConfig)
		t.FailNow()
	}
}

func TestAssignConfig_02(t *testing.T) {
	_BConfig := &Config{}
	bs, _ := json.Encode(newBConfig(), false)

	jsonMap := M{}
	_ = json.Decode(bs, &jsonMap)

	configMap := M{}
	for k, v := range jsonMap {
		if reflect.TypeOf(v).Kind() == reflect.Map {
			for k1, v1 := range v.(M) {
				if reflect.TypeOf(v1).Kind() == reflect.Map {
					for k2, v2 := range v1.(M) {
						configMap[k2] = v2
					}
				} else {
					configMap[k1] = v1
				}
			}
		} else {
			configMap[k] = v
		}
	}
	configMap["AppName"] = "asana"
	configMap["MaxMemory"] = 1024
	configMap["Graceful"] = true
	configMap["XSRFExpire"] = 32
	configMap["SessionProviderConfig"] = "file"
	configMap["FileLineNum"] = true

	bs, _ = json.Encode(configMap, false)

	jcf := config.NewConfig()

	_ = jcf.Load(memory.NewSource(memory.WithJson(bs)))

	for _, i := range []interface{}{_BConfig, &_BConfig.Listen, &_BConfig.WebConfig, &_BConfig.Log, &_BConfig.WebConfig.Session} {
		assignSingleConfig(i, jcf)
	}

	if _BConfig.MaxMemory != 1024 {
		t.Log(_BConfig.MaxMemory)
		t.FailNow()
	}

	if !_BConfig.Listen.Graceful {
		t.Log(_BConfig.Listen.Graceful)
		t.FailNow()
	}

	if _BConfig.WebConfig.XSRFExpire != 32 {
		t.Log(_BConfig.WebConfig.XSRFExpire)
		t.FailNow()
	}

	if _BConfig.WebConfig.Session.SessionProviderConfig != "file" {
		t.Log(_BConfig.WebConfig.Session.SessionProviderConfig)
		t.FailNow()
	}

	if !_BConfig.Log.FileLineNum {
		t.Log(_BConfig.Log.FileLineNum)
		t.FailNow()
	}

}
