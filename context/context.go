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
//	import "github.com/goasana/framework/context"
//
//	ctx := Context{HTTPRequest:req,ResponseWriter:rw}
//
//  more docs http://asana.me/docs/module/md
package context

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/beego/i18n"
	"github.com/goasana/config/encoder/json"
	"github.com/goasana/config/encoder/proto"
	"github.com/goasana/config/encoder/xml"
	"github.com/goasana/config/encoder/yaml"
	"github.com/goasana/framework/session"
	"github.com/goasana/framework/utils"
)

//commonly used mime-types
const (
	ApplicationHTML     = "application/xhtml+xml"
	ApplicationJSON     = "application/json"
	ApplicationJSONP    = "application/javascript"
	ApplicationXML      = "application/xml"
	ApplicationYAML     = "application/x-yaml"
	ApplicationProtoBuf = "application/x-protobuf"
	TextXML             = "text/xml"
	TextHTML            = "text/html"
	TextPlain           = "text/plain"
)

// NewContext return the Context with Request and Response
func NewContext() *Context {
	return &Context{
		Request:  NewRequest(),
		Response: NewResponse(),
		Data: map[interface{}]interface{}{},
	}
}

// Context Http request context struct including AsanaRequest, AsanaResponse, http.HTTPRequest and http.ResponseWriter.
// AsanaRequest and AsanaResponse provides some api to operate request and response more easily.
type Context struct {
	Data map[interface{}]interface{}

	Request        *AsanaRequest
	Response       *AsanaResponse
	HTTPRequest    *http.Request
	ResponseWriter *Response
	_xsrfToken     string

	// xsrf data
	XSRFExpire int
	EnableXSRF bool

	// session
	CruSession session.Store

	isPro bool
}

// SetData set the data depending on the accepted
func (ctx *Context) SetData(data interface{}) *Context {
	accept := ctx.Request.Header("Accept")
	switch accept {
	case ApplicationYAML:
		ctx.Data["yaml"] = data
	case ApplicationXML, TextXML:
		ctx.Data["xml"] = data
	case ApplicationProtoBuf:
		ctx.Data["protobuf"] = data
	case ApplicationJSONP:
		ctx.Data["jsonp"] = data
	case ApplicationJSON:
		ctx.Data["json"] = data
	case TextHTML:
		ctx.Data["html"] = data
	default:
		ctx.Data["txt"] = data
	}

	return ctx
}

// SetPro set if environment is production
func (ctx *Context) SetPro(isPro bool) {
	ctx.isPro = isPro
}

	// ServeJSON sends a json response with encoding charset.
func (ctx *Context) ServeJSON(encoding ...bool) error {
	hasIndent := !ctx.isPro
	hasEncoding := len(encoding) > 0 && encoding[0]
	return ctx.JSON(ctx.Data["json"], hasIndent, hasEncoding)
}

// ServeJSONP sends a jsonp response.
func (ctx *Context) ServeJSONP() error {
	hasIndent := !ctx.isPro
	return ctx.JSONP(ctx.Data["jsonp"], hasIndent)
}

// ServeXML sends xml response.
func (ctx *Context) ServeXML() error {
	hasIndent := !ctx.isPro
	return ctx.XML(ctx.Data["xml"], hasIndent)
}

// ServeYAML sends yaml response.
func (ctx *Context) ServeYAML() error {
	return ctx.YAML(ctx.Data["yaml"])
}

// ServeProtoBuf sends protobuf response.
func (ctx *Context) ServeProtoBuf() error {
	return ctx.ProtoBuf(ctx.Data["protobuf"])
}

// ServeHTML sends html response.
func (ctx *Context) ServeHTML() error {
	switch ctx.Data["html"].(type) {
	case string:
		val := ctx.Data["html"].(string)
		return ctx.HTML(val)
	case []byte:
		val := ctx.Data["html"].([]byte)
		return ctx.HTMLBlob(val)
	default:
		return errors.New("no data found")
	}
}

// ServeText sends text plain response.
func (ctx *Context) ServeText() error {
	var data interface{}
	if v, ok := ctx.Data["txt"]; ok {
		data = v
	} else {
		for _, v := range ctx.Data {
			data = v
			break
		}
	}

	switch data.(type) {
	case string:
		val := data.(string)
		return ctx.Text(val)
	case []byte:
		val := data.([]byte)
		return ctx.TextBlob(val)
	default:
		return errors.New("no data found")
	}
}

// ServeFormatted serve YAML, XML, JSON, ProtoBuffer, Html or Text, depending on the value of the Accept header
func (ctx *Context) ServeFormatted(encoding ...bool) error {
	accept := ctx.Request.Header("Accept")
	switch accept {
	case ApplicationYAML:
		return ctx.ServeYAML()
	case ApplicationXML, TextXML:
		return ctx.ServeXML()
	case ApplicationProtoBuf:
		return ctx.ServeProtoBuf()
	case ApplicationJSONP:
		return ctx.ServeJSONP()
	case ApplicationJSON:
		return ctx.ServeJSON(encoding...)
	case TextHTML:
		return ctx.ServeHTML()
	default:
		return ctx.ServeText()
	}
}

// Reset init Context, AsanaRequest and AsanaResponse
func (ctx *Context) Reset(rw http.ResponseWriter, r *http.Request) {
	ctx.HTTPRequest = r
	if ctx.ResponseWriter == nil {
		ctx.ResponseWriter = &Response{}
	}
	ctx.ResponseWriter.reset(rw)
	ctx.Request.Reset(ctx)
	ctx.Response.Reset(ctx)
	ctx._xsrfToken = ""
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
	if v := ctx.Request.Query(key); v != "" {
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
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.Atoi(strv)
}

// GetInt8 return input as an int8 or the default value while it's present and input is blank
func (ctx *Context) GetInt8(key string, def ...int8) (int8, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 8)
	return int8(i64), err
}

// GetUint8 return input as an uint8 or the default value while it's present and input is blank
func (ctx *Context) GetUint8(key string, def ...uint8) (uint8, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 8)
	return uint8(u64), err
}

// GetInt16 returns input as an int16 or the default value while it's present and input is blank
func (ctx *Context) GetInt16(key string, def ...int16) (int16, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 16)
	return int16(i64), err
}

// GetUint16 returns input as an uint16 or the default value while it's present and input is blank
func (ctx *Context) GetUint16(key string, def ...uint16) (uint16, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 16)
	return uint16(u64), err
}

// GetInt32 returns input as an int32 or the default value while it's present and input is blank
func (ctx *Context) GetInt32(key string, def ...int32) (int32, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 32)
	return int32(i64), err
}

// GetUint32 returns input as an uint32 or the default value while it's present and input is blank
func (ctx *Context) GetUint32(key string, def ...uint32) (uint32, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 32)
	return uint32(u64), err
}

// GetInt64 returns input value as int64 or the default value while it's present and input is blank.
func (ctx *Context) GetInt64(key string, def ...int64) (int64, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseInt(strv, 10, 64)
}

// GetUint64 returns input value as uint64 or the default value while it's present and input is blank.
func (ctx *Context) GetUint64(key string, def ...uint64) (uint64, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseUint(strv, 10, 64)
}

// GetBool returns input value as bool or the default value while it's present and input is blank.
func (ctx *Context) GetBool(key string, def ...bool) (bool, error) {
	strv := ctx.Request.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseBool(strv)
}

// GetFloat returns input value as float64 or the default value while it's present and input is blank.
func (ctx *Context) GetFloat(key string, def ...float64) (float64, error) {
	strv := ctx.Request.Query(key)
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

// SetStatus sets de code http
func (ctx *Context) SetStatus(code int) *Context {
	ctx.Response.Status = code
	return ctx
}

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
		ctx.CruSession = ctx.Request.CruSession
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
	return ctx.Request.IsAjax()
}


// XSRFFormHTML writes an input field contains xsrf token value.
func (ctx *Context) XSRFFormHTML() string {
	return `<input type="hidden" name="_xsrf" value="` +
		ctx._xsrfToken + `" />`
}

// Abort stops this request.
// if asana.ErrorMaps exists, panic body.
func (ctx *Context) Abort(body string) error {
	panic(body)
	return nil
}

// GetJWT get token
func (ctx *Context) GetJWT() (string, error) {
	authHeader := ctx.Request.Header(HeaderAuthorization)
	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || !isJWTHeader(authHeaderParts[0]) {
		return "", errors.New("authorization header format must be Bearer|JWT|Token {token}")
	}

	return authHeaderParts[1], nil
}

func isJWTHeader(header string) bool {
	for _, v := range strings.Fields("bearer jwt token") {
		if strings.ToLower(header) == v {
			return true
		}
	}

	return false
}


// GetLanguage get the language accepted
func (ctx *Context) GetLanguage(def ...string) string {
	al := ctx.Request.Header(HeaderAcceptLanguage)

	var lang string
	if len(def) > 0 {
		lang = def[0]
	}

	if len(al) > 0 {
		if len(al) > 4 {
			if i18n.IsExist(al[:5]) {
				lang = al[:5]
			} else if i18n.IsExist(al[:2]) {
				lang = al[:2]
			}
		}
	}

	return lang
}


// Text writes plain text to.Body.
func (ctx *Context) Text(data string) error {
	return ctx.TextBlob([]byte(data))
}

// TextBlob writes plain text to.Body from []byte.
func (ctx *Context) TextBlob(data []byte) error {
	return ctx.Blob(TextPlain, []byte(data))
}

// HTML writes html text to.Body.
func (ctx *Context) HTML(html string) error {
	return ctx.HTMLBlob([]byte(html))
}

// HTMLBlob writes html text to.Body from []byte.
func (ctx *Context) HTMLBlob(data []byte) error {
	return ctx.Blob(TextHTML, data)
}

// Blob writes []byte to.Body.
func (ctx *Context) Blob(contentType string, b []byte) error {
	return ctx.Header(HeaderContentType, getContentTypeHead(contentType)).Body(b)
}

// Stream writes stream to.Body.
func (ctx *Context) Stream(contentType string, r io.Reader) (err error) {
	ctx.Header(HeaderContentType, getContentTypeHead(contentType))
	_, err = io.Copy(ctx.ResponseWriter, r)
	return
}

// NoContent white response
func (ctx *Context) NoContent() error {
	ctx.ResponseWriter.WriteHeader(ctx.Response.Status)
	return nil
}

// Redirect header location redirect
func (ctx *Context) Redirect(url string) error {
	if !ctx.Response.IsRedirect() {
		return errors.New("invalid redirect status code")
	}

	ctx.ResponseWriter.Header().Set(HeaderLocation, url)
	ctx.ResponseWriter.WriteHeader(ctx.Response.Status)
	return nil
}

// JSON writes json to.Bodyresponse.
// if encoding is true, it converts utf-8 to \u0000 type.
func (ctx *Context) JSON(data interface{}, hasIndent bool, encoding bool) error {
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	if encoding {
		content = []byte(stringsToJSON(string(content)))
	}
	return ctx.Header(HeaderContentType, getContentTypeHead(ApplicationJSON)).Body(content)
}

// ProtoBuf writes protobuf to.Body.
func (ctx *Context) ProtoBuf(data interface{}) error {
	content, err := proto.Encode(data)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return ctx.Header(HeaderContentType, getContentTypeHead(ApplicationProtoBuf)).Body(content)
}

// YAML writes yaml to.Body.
func (ctx *Context) YAML(data interface{}) error {
	content, err := yaml.Encode(data)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return ctx.Header(HeaderContentType, getContentTypeHead(ApplicationYAML)).Body(content)
}

// JSONP writes jsonp to.Body.
func (ctx *Context) JSONP(data interface{}, hasIndent bool) error {
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := ctx.Request.Query("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback = template.JSEscapeString(callback)
	callbackContent := bytes.NewBufferString(" if(window." + callback + ")" + callback)
	callbackContent.WriteString("(")
	callbackContent.Write(content)
	callbackContent.WriteString(");\r\n")
	return ctx.Header(HeaderContentType, getContentTypeHead(ApplicationJSONP)).Body(callbackContent.Bytes())
}

// XML writes xml string to.Body.
func (ctx *Context) XML(data interface{}, hasIndent bool) error {
	content, err := xml.Encode(data, hasIndent)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return ctx.Header(HeaderContentType, getContentTypeHead(ApplicationXML)).Body(content)
}


// Header sets.Header item string via given key.
func (ctx *Context) Header(key, val string) *Context {
	ctx.ResponseWriter.Header().Set(key, val)
	return ctx
}

// Body sets.Body content.
// if EnableGzip, compress content string.
// it sends out.Body directly.
func (ctx *Context) Body(content []byte) error {
	var encoding string
	var buf = &bytes.Buffer{}
	if ctx.Response.EnableGzip {
		encoding = ParseEncoding(ctx.HTTPRequest)
	}
	if b, n, _ := WriteBody(encoding, buf, content); b {
		ctx.Header(HeaderContentEncoding, n)
		ctx.Header(HeaderContentLength, strconv.Itoa(buf.Len()))
	} else {
		ctx.Header(HeaderContentLength, strconv.Itoa(len(content)))
	}
	// Write status code if it has been set manually
	// Set it to 0 afterwards to prevent "multiple response.WriteHeader calls"
	if ctx.Response.Status != 0 {
		ctx.ResponseWriter.WriteHeader(ctx.Response.Status)
		ctx.Response.Status = 0
	} else {
		ctx.ResponseWriter.Started = true
	}
	_, _ = io.Copy(ctx.ResponseWriter, buf)
	return nil
}

// Download forces response for download file.
// it prepares the download.Header automatically.
func (ctx *Context) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(ctx.ResponseWriter, ctx.HTTPRequest, file)
		return
	}

	var fName string
	if len(filename) > 0 && filename[0] != "" {
		fName = filename[0]
	} else {
		fName = filepath.Base(file)
	}
	// https://tools.ietf.org/html/rfc6266#section-4.3
	fn := url.PathEscape(fName)
	if fName == fn {
		fn = "filename=" + fn
	} else {
		/**
		  The parameters "filename" and "filename*" differ only in that
		  "filename*" uses the encoding defined in [RFC5987], allowing the use
		  of characters not present in the ISO-8859-1 character set
		  ([ISO-8859-1]).
		*/
		fn = "filename=" + fName + "; filename*=utf-8''" + fn
	}
	ctx.Header(HeaderContentDisposition, "attachment; "+fn)
	ctx.Header(HeaderContentDescription, "File Transfer")
	ctx.Header(HeaderContentType, "application/octet-stream")
	ctx.Header(HeaderContentTransferEncoding, "binary")
	ctx.Header(HeaderExpires, "0")
	ctx.Header(HeaderCacheControl, "must-revalidate")
	ctx.Header(HeaderPragma, "public")
	http.ServeFile(ctx.ResponseWriter, ctx.HTTPRequest, file)
}

// ContentType sets the content type from ext string.
// MIME type is given in mime package.
func (ctx *Context) ContentType(ext string) *Context {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		ctx.Header(HeaderContentType, ctype)
	}
	return ctx
}

// WriteString Write string to.Body.
// it sends.Body.
func (ctx *Context) WriteString(content string) (err error) {
	_, err = ctx.ResponseWriter.Write([]byte(content))
	return
}

// GetCookie Get cookie from request by a given key.
// It's alias of AsanaRequest.Cookie.
func (ctx *Context) GetCookie(key string) string {
	return ctx.Request.Cookie(key)
}

// SetCookie Set cookie for response.
// It's alias of AsanaResponse.Cookie.
func (ctx *Context) SetCookie(name string, value string, others ...interface{}) {
	ctx.Response.Cookie(name, value, others...)
}

// GetSecureCookie Get secure cookie from request by a given key.
func (ctx *Context) GetSecureCookie(Secret, key string) (string, bool) {
	val := ctx.Request.Cookie(key)
	if val == "" {
		return "", false
	}

	parts := strings.SplitN(val, "|", 3)

	if len(parts) != 3 {
		return "", false
	}

	vs := parts[0]
	timestamp := parts[1]
	sig := parts[2]

	h := hmac.New(sha1.New, []byte(Secret))
	_, _ = fmt.Fprintf(h, "%s%s", vs, timestamp)

	if fmt.Sprintf("%02x", h.Sum(nil)) != sig {
		return "", false
	}
	res, _ := base64.URLEncoding.DecodeString(vs)
	return string(res), true
}

// SetSecureCookie Set Secure cookie for response.
func (ctx *Context) SetSecureCookie(Secret, name, value string, others ...interface{}) *Context {
	vs := base64.URLEncoding.EncodeToString([]byte(value))
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	h := hmac.New(sha1.New, []byte(Secret))
	_, _ = fmt.Fprintf(h, "%s%s", vs, timestamp)
	sig := fmt.Sprintf("%02x", h.Sum(nil))
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	ctx.Response.Cookie(name, cookie, others...)
	return ctx
}

// SetXSRFToken creates a xsrf token string and returns.
func (ctx *Context) SetXSRFToken(key string, expire int64) string {
	if ctx._xsrfToken == "" {
		token, ok := ctx.GetSecureCookie(key, "_xsrf")
		if !ok {
			token = string(utils.RandomCreateBytes(32))
			ctx.SetSecureCookie(key, "_xsrf", token, expire)
		}
		ctx._xsrfToken = token
	}
	return ctx._xsrfToken
}

// XSRFToken get _xsrfToken value
func (ctx *Context) XSRFToken() string {
	return ctx._xsrfToken
}

// CheckXSRFCookie checks xsrf token in this request is valid or not.
// the token can provided in request header "X-CsrfToken"
// or in form field value named as "_xsrf".
func (ctx *Context) CheckXSRFCookie() bool {
	if !ctx.EnableXSRF {
		return true
	}

	token := ctx.Request.Query("_xsrf")
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

// RenderMethodResult renders the return value of a controller method to the output
func (ctx *Context) RenderMethodResult(result interface{}) {
	if result != nil {
		renderer, ok := result.(Renderer)
		if !ok {
			err, ok := result.(error)
			if ok {
				renderer = errorRenderer(err)
			} else {
				renderer = jsonRenderer(result)
			}
		}
		renderer.Render(ctx)
	}
}

//Response is a wrapper for the http.ResponseWriter
//started set to true if response was written to then don't execute other handler
type Response struct {
	http.ResponseWriter
	Started bool
	Status  int
	Elapsed time.Duration
}

func (r *Response) reset(rw http.ResponseWriter) {
	r.ResponseWriter = rw
	r.Status = 0
	r.Started = false
}

// Write writes the data to the connection as part of an HTTP reply,
// and sets `started` to true.
// started means the response has sent out.
func (r *Response) Write(p []byte) (int, error) {
	r.Started = true
	return r.ResponseWriter.Write(p)
}

// WriteHeader sends an HTTP.Header with status code,
// and sets `started` to true.
func (r *Response) WriteHeader(code int) {
	if r.Status > 0 {
		//prevent multiple response.WriteHeader calls
		return
	}
	r.Status = code
	r.Started = true
	r.ResponseWriter.WriteHeader(code)
}

// Hijack hijacker for http
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("webserver doesn't support hijacking")
	}
	return hj.Hijack()
}

// Flush http.Flusher
func (r *Response) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Pusher http.Pusher
func (r *Response) Pusher() (pusher http.Pusher) {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
