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
	"bytes"
	"errors"
	"fmt"
	"github.com/goasana/asana/context/parsers"
	"html/template"
	"net/http"
	"reflect"
	"strings"

	"github.com/goasana/asana/context"
	"github.com/goasana/asana/context/param"
)

var (
	// ErrAbort custom error when user stop request handler manually.
	ErrAbort = errors.New("user stop run")
	// GlobalControllerRouter store comments with controller. pkgpath+controller:comments
	GlobalControllerRouter = make(map[string][]ControllerComments)
)

// ControllerFilter store the filter for controller
type ControllerFilter struct {
	Pattern        string
	Pos            int
	Filter         FilterFunc
	ReturnOnOutput bool
	ResetParams    bool
}

// ControllerFilterComments store the comment for controller level filter
type ControllerFilterComments struct {
	Pattern        string
	Pos            int
	Filter         string // NOQA
	ReturnOnOutput bool
	ResetParams    bool
}

// ControllerImportComments store the import comment for controller needed
type ControllerImportComments struct {
	ImportPath  string
	ImportAlias string
}

// ControllerComments store the comment for the controller method
type ControllerComments struct {
	Method           string
	Router           string
	Filters          []*ControllerFilter
	ImportComments   []*ControllerImportComments
	FilterComments   []*ControllerFilterComments
	AllowHTTPMethods []string
	Params           []map[string]string
	MethodParams     []*param.MethodParam
}

// ControllerCommentsSlice implements the sort interface
type ControllerCommentsSlice []ControllerComments

func (p ControllerCommentsSlice) Len() int           { return len(p) }
func (p ControllerCommentsSlice) Less(i, j int) bool { return p[i].Router < p[j].Router }
func (p ControllerCommentsSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Controller defines some basic http request handler operations, such as
// http context, template and view, session and xsrf.
type Controller struct {
	// context data
	*context.Context

	// route controller info
	controllerName string
	actionName     string
	methodMapping  map[string]func() //method:routertree
	AppController  interface{}

	// template data
	TplName        string
	ViewPath       string
	Layout         string
	LayoutSections map[string]string // the key is the section name and the value is the template name
	TplPrefix      string
	TplExt         string
	EnableRender   bool
}

// ControllerInterface is an interface to uniform all controller handler.
type ControllerInterface interface {
	Init(ct *context.Context, controllerName, actionName string, app interface{})
	Prepare()
	Get()
	Post()
	Delete()
	Put()
	Head()
	Patch()
	Options()
	Trace()
	Finish()
	Render() error
	XSRFToken() string
	CheckXSRFCookie() bool
	HandlerFunc(fn string) bool
	URLMapping()
}

// Init generates default values of controller operations.
func (c *Controller) Init(ctx *context.Context, controllerName, actionName string, app interface{}) {
	c.Layout = ""
	c.TplName = ""
	c.controllerName = controllerName
	c.actionName = actionName
	c.Context = ctx
	c.TplExt = "tpl"
	c.AppController = app
	c.EnableRender = true
	c.EnableXSRF = true
	c.methodMapping = make(map[string]func())
	c.IsPro = BConfig.RunMode == PROD
}

// Prepare runs after Init before request function execution.
func (c *Controller) Prepare() {}

// Finish runs after request function execution.
func (c *Controller) Finish() {}

// Get adds a request function to handle GET request.
func (c *Controller) Get() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Post adds a request function to handle POST request.
func (c *Controller) Post() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Delete adds a request function to handle DELETE request.
func (c *Controller) Delete() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Put adds a request function to handle PUT request.
func (c *Controller) Put() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Head adds a request function to handle HEAD request.
func (c *Controller) Head() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Patch adds a request function to handle PATCH request.
func (c *Controller) Patch() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Options adds a request function to handle OPTIONS request.
func (c *Controller) Options() {
	http.Error(c.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// Trace adds a request function to handle Trace request.
// this method SHOULD NOT be overridden.
// https://tools.ietf.org/html/rfc7231#section-4.3.8
// The TRACE method requests a remote, application-level loop-back of
// the request message.  The final recipient of the request SHOULD
// reflect the message received, excluding some fields described below,
// back to the client as the message body of a 200 (OK) response with a
// Content-Type of "message/http" (Section 8.3.1 of [RFC7230]).
func (c *Controller) Trace() {
	ts := func(h http.Header) (hs string) {
		for k, v := range h {
			hs += fmt.Sprintf("\r\n%s: %s", k, v)
		}
		return
	}
	hs := fmt.Sprintf("\r\nTRACE %s %s%s\r\n", c.HTTPRequest.RequestURI, c.HTTPRequest.Proto, ts(c.HTTPRequest.Header))
	c.SetHeader(context.HeaderContentType, "message/http")
	c.SetHeader(context.HeaderContentLength, fmt.Sprint(len(hs)))
	c.SetHeader(context.HeaderCacheControl, "no-cache, no-store, must-revalidate")
	_ = c.Text(hs)
}

// HandlerFunc call function with the name
func (c *Controller) HandlerFunc(fnName string) bool {
	if v, ok := c.methodMapping[fnName]; ok {
		v()
		return true
	}
	return false
}

// URLMapping register the internal Controller router.
func (c *Controller) URLMapping() {}

// Mapping the method to function
func (c *Controller) Mapping(method string, fn func()) {
	c.methodMapping[method] = fn
}

// Render sends the response with rendered template bytes as text/html type.
func (c *Controller) Render() error {
	if !c.EnableRender {
		return nil
	}
	rb, err := c.RenderBytes()
	if err != nil {
		return err
	}

	if c.ResponseWriter.Header().Get(context.HeaderContentType) == "" {
		c.SetHeader(context.HeaderContentType, "text/html; charset=utf-8")
	}

	return c.Body(rb)
}

// RenderString returns the rendered template string. Do not send out response.
func (c *Controller) RenderString() (string, error) {
	b, e := c.RenderBytes()
	return string(b), e
}

// RenderBytes returns the bytes of rendered template string. Do not send out response.
func (c *Controller) RenderBytes() ([]byte, error) {
	buf, err := c.renderTemplate()
	//if the controller has set layout, then first get the tplName's content set the content to the layout
	if err == nil && c.Layout != "" {
		c.Response().PutData("LayoutContent", template.HTML(buf.String()))

		if c.LayoutSections != nil {
			for sectionName, sectionTpl := range c.LayoutSections {
				if sectionTpl == "" {
					c.Response().PutData(sectionName, "")
					continue
				}
				buf.Reset()
				err = ExecuteViewPathTemplate(&buf, sectionTpl, c.viewPath(), c.Response().GetData())
				if err != nil {
					return nil, err
				}
				c.Response().PutData(sectionName, template.HTML(buf.String()))
			}
		}

		buf.Reset()
		_ = ExecuteViewPathTemplate(&buf, c.Layout, c.viewPath(), c.Response().GetData())
	}
	return buf.Bytes(), err
}

func (c *Controller) renderTemplate() (bytes.Buffer, error) {
	var buf bytes.Buffer
	if c.TplName == "" {
		c.TplName = strings.ToLower(c.controllerName) + "/" + strings.ToLower(c.actionName) + "." + c.TplExt
	}
	if c.TplPrefix != "" {
		c.TplName = c.TplPrefix + c.TplName
	}
	if BConfig.RunMode == DEV {
		buildFiles := []string{c.TplName}
		if c.Layout != "" {
			buildFiles = append(buildFiles, c.Layout)
			if c.LayoutSections != nil {
				for _, sectionTpl := range c.LayoutSections {
					if sectionTpl == "" {
						continue
					}
					buildFiles = append(buildFiles, sectionTpl)
				}
			}
		}
		_ = BuildTemplate(c.viewPath(), buildFiles...)
	}
	return buf, ExecuteViewPathTemplate(&buf, c.TplName, c.viewPath(), c.Context.Response().GetData())
}

func (c *Controller) viewPath() string {
	if c.ViewPath == "" {
		return BConfig.WebConfig.ViewsPath
	}
	return c.ViewPath
}

// Redirect sends the redirection response to url with status code.
func (c *Controller) Redirect(url string) error {
	LogAccess(c.Context, nil, c.GetStatus())
	return c.Context.Redirect(url)
}

// URLFor does another controller handler in this request function.
// it goes to this controller method if endpoint is not clear.
func (c *Controller) URLFor(endpoint string, values ...interface{}) string {
	if len(endpoint) == 0 {
		return ""
	}
	if endpoint[0] == '.' {
		return URLFor(reflect.Indirect(reflect.ValueOf(c.AppController)).Type().Name()+endpoint, values...)
	}
	return URLFor(endpoint, values...)
}

// ParseForm maps input data map to obj struct.
func (c *Controller) ParseForm(obj interface{}) error {
	return ParseForm(c.Input(), obj)
}

// ParseBody input data map to obj struct depending on the type of content.
func (c *Controller) ParseBody(obj interface{}) error {
	if c.AcceptsJSON() {
		return parsers.GetProvider(parsers.JSON).Parse(c.HTTPRequest, obj)
	} else if c.AcceptsYAML() {
		return parsers.GetProvider(parsers.YAML).Parse(c.HTTPRequest, obj)
	} else if c.AcceptsXML() {
		return parsers.GetProvider(parsers.XML).Parse(c.HTTPRequest, obj)
	} else if c.AcceptsProtoBuf() {
		return parsers.GetProvider(parsers.PROTOBUF).Parse(c.HTTPRequest, obj)
	}
	return c.ParseBody(obj)
}

// CustomAbort stops controller handler and show the error data, it's similar Aborts, but support status code and body.
func (c *Controller) CustomAbort(status int, body string) {
	// first panic from ErrorMaps, it is user defined error functions.
	c.Response().SetStatus(status)

	if _, ok := ErrorMaps[body]; ok {
		panic(body)
	}
	// last panic user string
	c.ResponseWriter.WriteHeader(status)
	_, _ = c.ResponseWriter.Write([]byte(body))
	panic(ErrAbort)
}

// StopRun makes panic of USERSTOPRUN error and go to recover function if defined.
func (c *Controller) StopRun() {
	panic(ErrAbort)
}

// SessionRegenerateID regenerates session id for this session.
// the session data have no changes.
func (c *Controller) SessionRegenerateID() {
	if c.CruSession != nil {
		c.CruSession.SessionRelease(c.ResponseWriter)
	}
	c.CruSession = GlobalSessions.SessionRegenerateID(c.ResponseWriter, c.HTTPRequest)
	c.CruSession = c.CruSession
}

// DestroySession cleans session data and session cookie.
func (c *Controller) DestroySession() {
	_ = c.CruSession.Flush()
	c.CruSession = nil
	GlobalSessions.SessionDestroy(c.ResponseWriter, c.HTTPRequest)
}

// GetXSRFToken creates a CSRF token string and returns.
func (c *Controller) GetXSRFToken() string {
	if c.XSRFToken() == "" {
		expire := int64(BConfig.WebConfig.XSRFExpire)
		if c.XSRFExpire > 0 {
			expire = int64(c.XSRFExpire)
		}
		c.SetXSRFToken(BConfig.WebConfig.XSRFKey, expire)
	}
	return c.XSRFToken()
}
