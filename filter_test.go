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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goasana/framework/context"
)

var FilterUser = func(ctx *context.Context) {
	_ = ctx.Response.Body([]byte("i am " + ctx.Request.Param(":last") + ctx.Request.Param(":first")))
}

func TestFilter(t *testing.T) {
	r, _ := http.NewRequest("GET", "/person/asa/na", nil)
	w := httptest.NewRecorder()
	handler := NewControllerRegister()
	_ = handler.InsertFilter("/person/:last/:first", BeforeRouter, FilterUser)
	handler.Add("/person/:last/:first", &TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am asana" {
		t.Errorf("user define func can't run")
	}
}

var FilterAdminUser = func(ctx *context.Context) {
	_ = ctx.Response.Body([]byte("i am admin"))
}

// Filter pattern /admin/:all
// all url like    /admin/    /admin/xie    will all get filter

func TestPatternTwo(t *testing.T) {
	r, _ := http.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler := NewControllerRegister()
	_ = handler.InsertFilter("/admin/?:all", BeforeRouter, FilterAdminUser)
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am admin" {
		t.Errorf("filter /admin/ can't run")
	}
}

func TestPatternThree(t *testing.T) {
	r, _ := http.NewRequest("GET", "/admin/asana", nil)
	w := httptest.NewRecorder()
	handler := NewControllerRegister()
	_ = handler.InsertFilter("/admin/:all", BeforeRouter, FilterAdminUser)
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am admin" {
		t.Errorf("filter /admin/asana can't run")
	}
}
