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
	"strings"
	"testing"

	"github.com/goasana/framework/context"
	"github.com/goasana/framework/logs"
	"github.com/goasana/framework/testdata/proto"
	"github.com/golang/protobuf/proto"
)

type TestController struct {
	Controller
}

func (tc *TestController) Get() {
	tc.Data["Username"] = "asana"
	_ = tc.Body([]byte("ok"))
}

func (tc *TestController) Post() {
	_ = tc.Body([]byte(tc.Request.Query(":name")))
}

func (tc *TestController) Param() {
	_ = tc.Body([]byte(tc.Request.Query(":name")))
}

func (tc *TestController) List() {
	_ = tc.Body([]byte("i am list"))
}

func (tc *TestController) Params() {
	_ = tc.Body([]byte(tc.Request.Param("0") + tc.Request.Param("1") + tc.Request.Param("2")))
}

func (tc *TestController) Myext() {
	_ = tc.Body([]byte(tc.Request.Param(":ext")))
}

func (tc *TestController) GetURL() {
	_ = tc.Body([]byte(tc.URLFor(".Myext")))
}

func (tc *TestController) GetParams() {
	_ = tc.WriteString(tc.Request.Query(":last") + "+" +
		tc.Request.Query(":first") + "+" + tc.Request.Query("learn"))
}

func (tc *TestController) GetManyRouter() {
	_ = tc.WriteString(tc.Request.Query(":id") + tc.Request.Query(":page"))
}

func (tc *TestController) GetEmptyBody() {
	_ = tc.NoContent()
}

type JSONController struct {
	Controller
}

func (jc *JSONController) Prepare() {
	jc.Data["json"] = "prepare"
	_ = jc.ServeJSON(true)
}

func (jc *JSONController) Get() {
	jc.Data["Username"] = "asana"
	_ = jc.Body([]byte("ok"))
}

func TestUrlFor(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/api/list", &TestController{}, "*:List")
	handler.Add("/person/:last/:first", &TestController{}, "*:Param")
	if a := handler.URLFor("TestController.List"); a != "/api/list" {
		logs.Info(a)
		t.Errorf("TestController.List must equal to /api/list")
	}
	if a := handler.URLFor("TestController.Param", ":last", "xie", ":first", "asta"); a != "/person/xie/asta" {
		t.Errorf("TestController.Param must equal to /person/xie/asta, but get " + a)
	}
}

func TestUrlFor3(t *testing.T) {
	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	if a := handler.URLFor("TestController.Myext"); a != "/test/myext" && a != "/Test/Myext" {
		t.Errorf("TestController.Myext must equal to /test/myext, but get " + a)
	}
	if a := handler.URLFor("TestController.GetURL"); a != "/test/geturl" && a != "/Test/GetURL" {
		t.Errorf("TestController.GetURL must equal to /test/geturl, but get " + a)
	}
}

func TestUrlFor2(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/v1/:v/cms_:id(.+)_:page(.+).html", &TestController{}, "*:List")
	handler.Add("/v1/:username/edit", &TestController{}, "get:GetURL")
	handler.Add("/v1/:v(.+)_cms/ttt_:id(.+)_:page(.+).html", &TestController{}, "*:Param")
	handler.Add("/:year:int/:month:int/:title/:entid", &TestController{})
	if handler.URLFor("TestController.GetURL", ":username", "asana") != "/v1/asana/edit" {
		logs.Info(handler.URLFor("TestController.GetURL"))
		t.Errorf("TestController.List must equal to /v1/asana/edit")
	}

	if handler.URLFor("TestController.List", ":v", "za", ":id", "12", ":page", "123") !=
		"/v1/za/cms_12_123.html" {
		logs.Info(handler.URLFor("TestController.List"))
		t.Errorf("TestController.List must equal to /v1/za/cms_12_123.html")
	}
	if handler.URLFor("TestController.Param", ":v", "za", ":id", "12", ":page", "123") !=
		"/v1/za_cms/ttt_12_123.html" {
		logs.Info(handler.URLFor("TestController.Param"))
		t.Errorf("TestController.List must equal to /v1/za_cms/ttt_12_123.html")
	}
	if handler.URLFor("TestController.Get", ":year", "1111", ":month", "11",
		":title", "aaaa", ":entid", "aaaa") !=
		"/1111/11/aaaa/aaaa" {
		logs.Info(handler.URLFor("TestController.Get"))
		t.Errorf("TestController.Get must equal to /1111/11/aaaa/aaaa")
	}
}

func TestUserFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/api/list", &TestController{}, "*:List")
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestPostFunc(t *testing.T) {
	r, _ := http.NewRequest("POST", "/asana", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/:name", &TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "asana" {
		t.Errorf("post func should asana")
	}
}

func TestAutoFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoFunc2(t *testing.T) {
	r, _ := http.NewRequest("GET", "/Test/List", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoFuncParams(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/params/2009/11/12", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "20091112" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoExtFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/myext.json", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "json" {
		t.Errorf("user define func can't run")
	}
}

func TestRouteOk(t *testing.T) {

	r, _ := http.NewRequest("GET", "/person/anderson/thomas?learn=kungfu", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/person/:last/:first", &TestController{}, "get:GetParams")
	handler.ServeHTTP(w, r)
	body := w.Body.String()
	if body != "anderson+thomas+kungfu" {
		t.Errorf("url param set to [%s];", body)
	}
}

func TestManyRoute(t *testing.T) {

	r, _ := http.NewRequest("GET", "/asana32-12.html", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/asana:id([0-9]+)-:page([0-9]+).html", &TestController{}, "get:GetManyRouter")
	handler.ServeHTTP(w, r)

	body := w.Body.String()

	if body != "3212" {
		t.Errorf("url param set to [%s];", body)
	}
}

// Test for issue #1669
func TestEmptyResponse(t *testing.T) {

	r, _ := http.NewRequest("GET", "/asana-empty.html", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/asana-empty.html", &TestController{}, "get:GetEmptyBody")
	handler.ServeHTTP(w, r)

	if body := w.Body.String(); body != "" {
		t.Error("want empty body")
	}
}

func TestNotFound(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Code set to [%v]; want [%v]", w.Code, http.StatusNotFound)
	}
}

// TestStatic tests the ability to serve static
// content from the filesystem
func TestStatic(t *testing.T) {
	r, _ := http.NewRequest("GET", "/static/js/jquery.js", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Errorf("handler.Static failed to serve file")
	}
}

func TestPrepare(t *testing.T) {
	r, _ := http.NewRequest("GET", "/json/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/json/list", &JSONController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != `"prepare"` {
		t.Errorf(w.Body.String() + "user define func can't run")
	}
}

func TestAutoPrefix(t *testing.T) {
	r, _ := http.NewRequest("GET", "/admin/test/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAutoPrefix("/admin", &TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("TestAutoPrefix can't run")
	}
}

func TestRouterGet(t *testing.T) {
	r, _ := http.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Get("/user", func(ctx *context.Context) {
		_ = ctx.Body([]byte("Get userlist"))
	})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "Get userlist" {
		t.Errorf("TestRouterGet can't run")
	}
}

func TestRouterPost(t *testing.T) {
	r, _ := http.NewRequest("POST", "/user/123", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Post("/user/:id", func(ctx *context.Context) {
		_ = ctx.Body([]byte(ctx.Request.Param(":id")))
	})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "123" {
		t.Errorf("TestRouterPost can't run")
	}
}

func sayhello(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("sayhello"))
}

func TestRouterHandler(t *testing.T) {
	r, _ := http.NewRequest("POST", "/sayhi", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Handler("/sayhi", http.HandlerFunc(sayhello))
	handler.ServeHTTP(w, r)
	if w.Body.String() != "sayhello" {
		t.Errorf("TestRouterHandler can't run")
	}
}

func TestRouterHandlerAll(t *testing.T) {
	r, _ := http.NewRequest("POST", "/sayhi/a/b/c", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Handler("/sayhi", http.HandlerFunc(sayhello), true)
	handler.ServeHTTP(w, r)
	if w.Body.String() != "sayhello" {
		t.Errorf("TestRouterHandler can't run")
	}
}

//
// Benchmarks NewApp:
//

func asanaFilterFunc(ctx *context.Context) {
	_ = ctx.WriteString("hello")
}

type AdminController struct {
	Controller
}

func (a *AdminController) Get() {
	_ = a.WriteString("hello")
}

func TestRouterFunc(t *testing.T) {
	mux := NewControllerRegister()
	mux.Get("/action", asanaFilterFunc)
	mux.Post("/action", asanaFilterFunc)
	rw, r := testRequest("GET", "/action")
	mux.ServeHTTP(rw, r)
	if rw.Body.String() != "hello" {
		t.Errorf("TestRouterFunc can't run")
	}
}

func BenchmarkFunc(b *testing.B) {
	mux := NewControllerRegister()
	mux.Get("/action", asanaFilterFunc)
	rw, r := testRequest("GET", "/action")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rw, r)
	}
}

func BenchmarkController(b *testing.B) {
	mux := NewControllerRegister()
	mux.Add("/action", &AdminController{})
	rw, r := testRequest("GET", "/action")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rw, r)
	}
}

func testRequest(method, path string) (*httptest.ResponseRecorder, *http.Request) {
	request, _ := http.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()

	return recorder, request
}

// Expectation: A Filter with the correct configuration should be created given
// specific parameters.
func TestInsertFilter(t *testing.T) {
	testName := "TestInsertFilter"

	mux := NewControllerRegister()
	_ = mux.InsertFilter("*", BeforeRouter, func(*context.Context) {})
	if !mux.filters[BeforeRouter][0].returnOnOutput {
		t.Errorf(
			"%s: passing no variadic params should set returnOnOutput to true",
			testName)
	}
	if mux.filters[BeforeRouter][0].resetParams {
		t.Errorf(
			"%s: passing no variadic params should set resetParams to false",
			testName)
	}

	mux = NewControllerRegister()
	_ = mux.InsertFilter("*", BeforeRouter, func(*context.Context) {}, false)
	if mux.filters[BeforeRouter][0].returnOnOutput {
		t.Errorf(
			"%s: passing false as 1st variadic param should set returnOnOutput to false",
			testName)
	}

	mux = NewControllerRegister()
	_ = mux.InsertFilter("*", BeforeRouter, func(*context.Context) {}, true, true)
	if !mux.filters[BeforeRouter][0].resetParams {
		t.Errorf(
			"%s: passing true as 2nd variadic param should set resetParams to true",
			testName)
	}
}

// Expectation: the second variadic arg should cause the execution of the filter
// to preserve the parameters from before its execution.
func TestParamResetFilter(t *testing.T) {
	testName := "TestParamResetFilter"
	route := "/asana/*" // splat
	path := "/asana/routes/routes"

	mux := NewControllerRegister()

	_ = mux.InsertFilter("*", BeforeExec, asanaResetParams, true, true)

	mux.Get(route, asanaHandleResetParams)

	rw, r := testRequest("GET", path)
	mux.ServeHTTP(rw, r)

	// The two functions, `asanaResetParams` and `asanaHandleResetParams` add
	// a.Header of `Splat`.  The expectation here is that that Header
	// value should match what the _request's_ router set, not the filter's.

	headers := rw.Result().Header
	if len(headers["Splat"]) != 1 {
		t.Errorf(
			"%s: There was an error in the test. Splat param not set in Header",
			testName)
	}
	if headers["Splat"][0] != "routes/routes" {
		t.Errorf(
			"%s: expected `:splat` param to be [routes/routes] but it was [%s]",
			testName, headers["Splat"][0])
	}
}

// Execution point: BeforeRouter
// expectation: only BeforeRouter function is executed, notmatch output as router doesn't handle
func TestFilterBeforeRouter(t *testing.T) {
	testName := "TestFilterBeforeRouter"
	url := "/beforeRouter"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, BeforeRouter, asanaBeforeRouter1)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "BeforeRouter1") {
		t.Errorf(testName + " BeforeRouter did not run")
	}
	if strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " BeforeRouter did not return properly")
	}
}

// Execution point: BeforeExec
// expectation: only BeforeExec function is executed, match as router determines route only
func TestFilterBeforeExec(t *testing.T) {
	testName := "TestFilterBeforeExec"
	url := "/beforeExec"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, BeforeRouter, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, BeforeExec, asanaBeforeExec1)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "BeforeExec1") {
		t.Errorf(testName + " BeforeExec did not run")
	}
	if strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " BeforeExec did not return properly")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
}

// Execution point: AfterExec
// expectation: only AfterExec function is executed, match as router handles
func TestFilterAfterExec(t *testing.T) {
	testName := "TestFilterAfterExec"
	url := "/afterExec"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, BeforeRouter, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, BeforeExec, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, AfterExec, asanaAfterExec1, false)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "AfterExec1") {
		t.Errorf(testName + " AfterExec did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeExec") {
		t.Errorf(testName + " BeforeExec ran in error")
	}
}

// Execution point: FinishRouter
// expectation: only FinishRouter function is executed, match as router handles
func TestFilterFinishRouter(t *testing.T) {
	testName := "TestFilterFinishRouter"
	url := "/finishRouter"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, BeforeRouter, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, BeforeExec, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, AfterExec, asanaFilterNoOutput)
	_ = mux.InsertFilter(url, FinishRouter, asanaFinishRouter1)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if strings.Contains(rw.Body.String(), "AfterExec1") {
		t.Errorf(testName + " AfterExec ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeExec") {
		t.Errorf(testName + " BeforeExec ran in error")
	}
}

// Execution point: FinishRouter
// expectation: only first FinishRouter function is executed, match as router handles
func TestFilterFinishRouterMultiFirstOnly(t *testing.T) {
	testName := "TestFilterFinishRouterMultiFirstOnly"
	url := "/finishRouterMultiFirstOnly"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, FinishRouter, asanaFinishRouter1, false)
	_ = mux.InsertFilter(url, FinishRouter, asanaFinishRouter2)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter1 did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	// not expected in body
	if strings.Contains(rw.Body.String(), "FinishRouter2") {
		t.Errorf(testName + " FinishRouter2 did run")
	}
}

// Execution point: FinishRouter
// expectation: both FinishRouter functions execute, match as router handles
func TestFilterFinishRouterMulti(t *testing.T) {
	testName := "TestFilterFinishRouterMulti"
	url := "/finishRouterMulti"

	mux := NewControllerRegister()
	_ = mux.InsertFilter(url, FinishRouter, asanaFinishRouter1, false)
	_ = mux.InsertFilter(url, FinishRouter, asanaFinishRouter2, false)

	mux.Get(url, asanaFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter1 did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if !strings.Contains(rw.Body.String(), "FinishRouter2") {
		t.Errorf(testName + " FinishRouter2 did not run properly")
	}
}

func asanaFilterNoOutput(_ *context.Context) {
}

func asanaBeforeRouter1(ctx *context.Context) {
	_ = ctx.WriteString("|BeforeRouter1")
}

func asanaBeforeExec1(ctx *context.Context) {
	_ = ctx.WriteString("|BeforeExec1")
}

func asanaAfterExec1(ctx *context.Context) {
	_ = ctx.WriteString("|AfterExec1")
}

func asanaFinishRouter1(ctx *context.Context) {
	_ = ctx.WriteString("|FinishRouter1")
}

func asanaFinishRouter2(ctx *context.Context) {
	_ = ctx.WriteString("|FinishRouter2")
}

func asanaResetParams(ctx *context.Context) {
	ctx.ResponseWriter.Header().Set("splat", ctx.Request.Param(":splat"))
}

func asanaHandleResetParams(ctx *context.Context) {
	ctx.ResponseWriter.Header().Set("splat", ctx.Request.Param(":splat"))
}

// YAML
type YAMLController struct {
	Controller
}

func (jc *YAMLController) Prepare() {
	jc.Data["yaml"] = "prepare"
	_ = jc.ServeYAML()
}

func (jc *YAMLController) Get() {
	jc.Data["Username"] = "asana"
	_ = jc.Body([]byte("ok"))
}

func TestYAMLPrepare(t *testing.T) {
	r, _ := http.NewRequest("GET", "/yaml/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/yaml/list", &YAMLController{})
	handler.ServeHTTP(w, r)
	if strings.TrimSpace(w.Body.String()) != "prepare" {
		t.Errorf(w.Body.String())
	}
}

// ProtoBuf
type ProtoBufController struct {
	Controller
}

var expectedProtoObject = &protoexample.Test{
	Name: "asana protobuf",
}

func (jc *ProtoBufController) Prepare() {
	jc.Data["protobuf"] = expectedProtoObject
	_ = jc.ServeProtoBuf()
}

func TestProtoBufPrepare(t *testing.T) {
	r, _ := http.NewRequest("GET", "/protobuf/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/protobuf/list", &ProtoBufController{})
	handler.ServeHTTP(w, r)
	res := strings.TrimSpace(w.Body.String())

	expectedBytes, _ := proto.Marshal(expectedProtoObject)

	if res != strings.TrimSpace(string(expectedBytes)) {
		t.Errorf(res)
	}
}
