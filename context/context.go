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

// Package context provide the context utils
// Usage:
//
//	import "github.com/goasana/asana/context"
//
//	ctx := Context{HTTPRequest:req,ResponseWriter:rw}
//
//  more docs http://asana.me/docs/module/md
package context

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/goasana/asana/session"
)

//commonly used mime-types
const (
	ApplicationHTML     = "application/xhtml+xml"
	ApplicationJSON     = "application/json"
	ApplicationJSONP    = "application/javascript"
	ApplicationXML      = "application/xml"
	ApplicationYAML     = "application/x-yaml"
	ApplicationProtoBuf = "application/x-protobuf"
	ApplicationMSGPack  = "application/x-msgpack"
	TextXML             = "text/xml"
	TextHTML            = "text/html"
	TextPlain           = "text/plain"
)

// Headers
const (
	HeaderAccept                  = "Accept"
	HeaderReferer                 = "Referer"
	HeaderUserAgent               = "User-Agent"
	HeaderAcceptEncoding          = "Accept-Encoding"
	HeaderAcceptLanguage          = "Accept-Language"
	HeaderExpires                 = "Expires"
	HeaderAllow                   = "Allow"
	HeaderAuthorization           = "Authorization"
	HeaderCacheControl            = "Cache-Control"
	HeaderPragma                  = "Pragma"
	HeaderContentDisposition      = "Content-Disposition"
	HeaderContentDescription      = "Content-Description"
	HeaderContentEncoding         = "Content-Encoding"
	HeaderContentTransferEncoding = "Content-Transfer-Encoding"
	HeaderContentLength           = "Content-Length"
	HeaderContentType             = "Content-Type"
	HeaderCookie                  = "Cookie"
	HeaderSetCookie               = "Set-Cookie"
	HeaderIfModifiedSince         = "If-Modified-Since"
	HeaderLastModified            = "Last-Modified"
	HeaderLocation                = "Location"
	HeaderUpgrade                 = "Upgrade"
	HeaderVary                    = "Vary"
	HeaderWWWAuthenticate         = "WWW-Authenticate"
	HeaderXForwardedFor           = "X-Forwarded-For"
	HeaderXForwardedProto         = "X-Forwarded-Proto"
	HeaderXForwardedProtocol      = "X-Forwarded-Protocol"
	HeaderXForwardedSsl           = "X-Forwarded-Ssl"
	HeaderXUrlScheme              = "X-Url-Scheme"
	HeaderXHTTPMethodOverride     = "X-HTTP-Method-Override"
	HeaderXRealIP                 = "X-Real-IP"
	HeaderXRequestID              = "X-Request-ID"
	HeaderXRequestedWith          = "X-Requested-With"
	HeaderServer                  = "Server"
	HeaderOrigin                  = "Origin"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXXSSProtection                  = "X-XSS-Protection"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderXCSRFToken                      = "X-CSRF-Token"
)

// NewContext return the Context with Request and ResponseWriter
func NewContext() *Context {
	return &Context{
		asanaRequest:  NewRequest(),
		asanaResponse: NewResponse(),
	}
}

// Context Http request context struct including asanaRequest, asanaResponse, http.HTTPRequest and http.ResponseWriter.
// asanaRequest and asanaResponse provides some api to operate request and response more easily.
type Context struct {
	*asanaRequest
	*asanaResponse

	CruSession session.Store
	// xsrf data
	XSRFExpire int
	EnableXSRF bool

	IsPro bool
}

func (ctx *Context) Request() Request {
	return ctx.asanaRequest
}

func (ctx *Context) Response() Response {
	return ctx.asanaResponse
}

// Session returns current session item value by a given key.
// if non-existed, return nil.
func (ctx *Context) Session(key interface{}) interface{} {
	return ctx.CruSession.Get(key)
}

// Reset init Context, asanaRequest and asanaResponse
func (ctx *Context) Reset(rw http.ResponseWriter, r *http.Request) {
	ctx.HTTPRequest = r
	if ctx.ResponseWriter == nil {
		ctx.ResponseWriter = &ResponseWriter{}
	}
	ctx.CruSession = nil
	ctx.ResponseWriter.reset(rw)
	ctx.Request().Reset(ctx)
	ctx.Response().Reset(ctx)
}

// Input Request returns the input data map from POST or PUT request body and query string.
func (ctx *Context) Input() url.Values {
	if ctx.HTTPRequest.Form == nil {
		_ = ctx.HTTPRequest.ParseForm()
	}
	return ctx.HTTPRequest.Form
}

// GetString returns the input value by key string or the default value while it's present and input is blank
func (ctx *Context) GetString(key string, def ...string) string {
	if v := ctx.Query(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// GetStrings returns the input string slice by key string or the default value while it's present and input is blank
// it's designed for multi-value input field such as checkbox(input[type=checkbox]), multi-selection.
func (ctx *Context) GetStrings(key string, def ...[]string) []string {
	var defv []string
	if len(def) > 0 {
		defv = def[0]
	}

	if f := ctx.Input(); f == nil {
		return defv
	} else if vs := f[key]; len(vs) > 0 {
		return vs
	}

	return defv
}

// GetInt returns input as an int or the default value while it's present and input is blank
func (ctx *Context) GetInt(key string, def ...int) (int, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.Atoi(strv)
}

// GetInt8 return input as an int8 or the default value while it's present and input is blank
func (ctx *Context) GetInt8(key string, def ...int8) (int8, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 8)
	return int8(i64), err
}

// GetUint8 return input as an uint8 or the default value while it's present and input is blank
func (ctx *Context) GetUint8(key string, def ...uint8) (uint8, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 8)
	return uint8(u64), err
}

// GetInt16 returns input as an int16 or the default value while it's present and input is blank
func (ctx *Context) GetInt16(key string, def ...int16) (int16, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 16)
	return int16(i64), err
}

// GetUint16 returns input as an uint16 or the default value while it's present and input is blank
func (ctx *Context) GetUint16(key string, def ...uint16) (uint16, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 16)
	return uint16(u64), err
}

// GetInt32 returns input as an int32 or the default value while it's present and input is blank
func (ctx *Context) GetInt32(key string, def ...int32) (int32, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 32)
	return int32(i64), err
}

// GetUint32 returns input as an uint32 or the default value while it's present and input is blank
func (ctx *Context) GetUint32(key string, def ...uint32) (uint32, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 32)
	return uint32(u64), err
}

// GetInt64 returns input value as int64 or the default value while it's present and input is blank.
func (ctx *Context) GetInt64(key string, def ...int64) (int64, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseInt(strv, 10, 64)
}

// GetUint64 returns input value as uint64 or the default value while it's present and input is blank.
func (ctx *Context) GetUint64(key string, def ...uint64) (uint64, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseUint(strv, 10, 64)
}

// GetBool returns input value as bool or the default value while it's present and input is blank.
func (ctx *Context) GetBool(key string, def ...bool) (bool, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseBool(strv)
}

// GetFloat returns input value as float64 or the default value while it's present and input is blank.
func (ctx *Context) GetFloat(key string, def ...float64) (float64, error) {
	strv := ctx.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseFloat(strv, 64)
}

// GetFile returns the file data in file upload field named as key.
// it returns the first one of multi-uploaded files.
func (ctx *Context) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.HTTPRequest.FormFile(key)
}

//GetFiles return multi-upload files
//files, err:=c.GetFiles("myfiles")
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusNoContent)
//		return
//	}
//for i, _ := range files {
//	//for each fileheader, get a handle to the actual file
//	file, err := files[i].Open()
//	defer file.Close()
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	//create destination file making sure the path is writeable.
//	dst, err := os.Create("upload/" + files[i].Filename)
//	defer dst.Close()
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	//copy the uploaded file to the destination file
//	if _, err := io.Copy(dst, file); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//}
// GetFiles get files
func (ctx *Context) GetFiles(key string) ([]*multipart.FileHeader, error) {
	if files, ok := ctx.HTTPRequest.MultipartForm.File[key]; ok {
		return files, nil
	}
	return nil, http.ErrMissingFile
}

// SaveToFile saves uploaded file to new path.
// it only operates the first one of mutil-upload form file field.
func (ctx *Context) SaveToFile(fromFile, toFile string) error {
	file, _, err := ctx.HTTPRequest.FormFile(fromFile)
	if err != nil {
		return err
	}
	defer file.Close()
	f, err := os.OpenFile(toFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, _ = io.Copy(f, file)
	return nil
}

// StartSession starts session and load old session data info this controller.
func (ctx *Context) StartSession() session.Store {
	if ctx.CruSession == nil {
		ctx.CruSession = ctx.CruSession
	}
	return ctx.CruSession
}

// SetSession puts value into session.
func (ctx *Context) SetSession(name interface{}, value interface{}) *Context {
	if ctx.CruSession == nil {
		ctx.StartSession()
	}
	_ = ctx.CruSession.Set(name, value)
	return ctx
}

// GetSession gets value from session.
func (ctx *Context) GetSession(name interface{}) interface{} {
	if ctx.CruSession == nil {
		ctx.StartSession()
	}
	return ctx.CruSession.Get(name)
}

// DelSession removes value from session.
func (ctx *Context) DelSession(name interface{}) {
	if ctx.CruSession == nil {
		ctx.StartSession()
	}
	_ = ctx.CruSession.Delete(name)
}

// IsAjax returns this request is ajax or not.
func (ctx *Context) IsAjax() bool {
	return ctx.IsAjax()
}

// XSRFFormHTML writes an input field contains xsrf token value.
func (ctx *Context) XSRFFormHTML() string {
	return `<input type="hidden" name="_xsrf" value="` +
		ctx._xsrfToken + `" />`
}

// ContentType sets the content type from ext string.
// MIME type is given in mime package.
func (ctx *Context) ContentType(ext string) *Context {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		ctx.SetHeader(HeaderContentType, ctype)
	}
	return ctx
}

// CheckXSRFCookie checks xsrf token in this request is valid or not.
// the token can provided in request header "X-CsrfToken"
// or in form field value named as "_xsrf".
func (ctx *Context) CheckXSRFCookie() bool {
	if !ctx.EnableXSRF {
		return true
	}

	token := ctx.Query("_xsrf")
	if token == "" {
		token = ctx.HTTPRequest.Header.Get(HeaderXCSRFToken)
	}
	if token == "" {
		_ = ctx.SetStatus(403).Abort("'_xsrf' argument missing from POST")
		return false
	}
	if ctx._xsrfToken != token {
		_ = ctx.SetStatus(403).Abort("XSRF cookie does not match POST argument")
		return false
	}
	return true
}
