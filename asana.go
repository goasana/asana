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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goasana/config/encoder"
	"github.com/goasana/config/source"
	"github.com/goasana/config/source/file"
)

const (
	// VERSION represent asana web framework version.
	VERSION = "1.13.1"

	// DEV is for develop
	DEV = "dev"
	// PROD is for production
	PROD = "prod"
)

// M is Map shortcut
type M map[string]interface{}

// Hook function to run
type hookFunc func() error

var (
	hooks = make([]hookFunc, 0) //hook function slice to store the hookFunc
)

// AddAPPStartHook is used to register the hookFunc
// The hookfuncs will run in asana.Run()
// such as initiating session , starting middleware , building template, starting admin control and so on.
func AddAPPStartHook(hf ...hookFunc) {
	hooks = append(hooks, hf...)
}

// Run asana application.
// asana.Run() default run on HttpPort
// asana.Run("localhost")
// asana.Run(":8089")
// asana.Run("127.0.0.1:8089")
func Run(params ...string) {

	initBeforeHTTPRun()

	if len(params) > 0 && params[0] != "" {
		strs := strings.Split(params[0], ":")
		if len(strs) > 0 && strs[0] != "" {
			BConfig.Listen.HTTPAddr = strs[0]
		}
		if len(strs) > 1 && strs[1] != "" {
			BConfig.Listen.HTTPPort, _ = strconv.Atoi(strs[1])
		}

		BConfig.Listen.Domains = params
	}

	AsanaApp.Run()
}

// RunWithMiddleWares Run asana application with middlewares.
func RunWithMiddleWares(addr string, mws ...MiddleWare) {
	initBeforeHTTPRun()

	strs := strings.Split(addr, ":")
	if len(strs) > 0 && strs[0] != "" {
		BConfig.Listen.HTTPAddr = strs[0]
		BConfig.Listen.Domains = []string{strs[0]}
	}
	if len(strs) > 1 && strs[1] != "" {
		BConfig.Listen.HTTPPort, _ = strconv.Atoi(strs[1])
	}

	AsanaApp.Run(mws...)
}

func initBeforeHTTPRun() {
	//init hooks
	AddAPPStartHook(
		registerMime,
		registerDefaultErrorHandler,
		registerSession,
		registerTemplate,
		registerAdmin,
		registerGzip,
	)

	for _, hk := range hooks {
		if err := hk(); err != nil {
			panic(err)
		}
	}
}

// TestAsanaInit is for test package init
func TestAsanaInit(ap string) {
	path := filepath.Join(ap, "conf", "app.json")
	_ = os.Chdir(ap)
	InitAsanaBeforeTest(path)
}

// InitAsanaBeforeTest is for test package init
func InitAsanaBeforeTest(appConfigPath string) {
	if err := LoadAppConfig(file.NewSource(file.WithPath(appConfigPath),
		source.WithEncoder(
			encoder.GetEncoder(strings.Replace(filepath.Ext(appConfigPath), ".", "", 1)))),
	); err != nil {
		panic(err)
	}
	BConfig.RunMode = "test"
	initBeforeHTTPRun()
}
