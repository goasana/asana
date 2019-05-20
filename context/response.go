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
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goasana/asana/utils"
	"github.com/goasana/config/encoder/json"
	"github.com/goasana/config/encoder/proto"
	"github.com/goasana/config/encoder/xml"
	"github.com/goasana/config/encoder/yaml"
)

// Response actions for response
type Response interface {
	Reset(res *Context) Response
	SetCookie(name string, value string, others ...interface{}) Response
	GetStatus() int
	SetStatus(status int) Response
	IsCachable() bool
	IsEmpty() bool
	IsOk() bool
	IsSuccessful() bool
	IsRedirect() bool
	IsForbidden() bool
	IsNotFound() bool
	IsClientError() bool
	IsServerError() bool
	JSON(data interface{}, hasIndent bool, encoding bool) error
	ProtoBuf(data interface{}) error
	YAML(data interface{}) error
	JSONP(data interface{}, hasIndent bool) error
	XML(data interface{}, hasIndent bool) error
	SetXSRFToken(key string, expire int64) string
	GetSecureCookie(Secret, key string) (string, bool)
	SetSecureCookie(Secret, name, value string, others ...interface{}) Response
	RenderMethodResult(result interface{}) Response
	Download(file string, filename ...string)
	SetBody(data interface{}) Response
	PutData(key interface{}, data interface{}) Response
	SetData(data map[interface{}]interface{}) Response
	GetData() map[interface{}]interface{}
	Abort(body string) error
	Redirect(url string) error
	Body(content []byte) error
	NoContent() error
	Text(data string) error
	TextBlob(data []byte) error
	HTML(html string) error
	HTMLBlob(data []byte) error
	Blob(contentType string, b []byte) error
	Stream(contentType string, r io.Reader) error
	ServeFormatted(encoding ...bool) error
}

// asanaResponse does work for sending.Header.
type asanaResponse struct {
	data           map[interface{}]interface{}
	ResponseWriter *ResponseWriter
	Context        *Context
	status         int
	EnableGzip     bool
	_xsrfToken     string
}

var _ Response = (*asanaResponse)(nil)

// newResponse returns new asanaResponse.
// it contains nothing now.
func newResponse() *asanaResponse {
	return &asanaResponse{
		data: make(map[interface{}]interface{}),
	}
}

// ServeJSON sends a json response with encoding charset.
func (res *asanaResponse) ServeJSON(encoding ...bool) error {
	hasIndent := !res.Context.IsPro
	hasEncoding := len(encoding) > 0 && encoding[0]
	return res.JSON(res.data["json"], hasIndent, hasEncoding)
}

// ServeJSONP sends a jsonp response.
func (res *asanaResponse) ServeJSONP() error {
	hasIndent := !res.Context.IsPro
	return res.JSONP(res.data["jsonp"], hasIndent)
}

// ServeXML sends xml response.
func (res *asanaResponse) ServeXML() error {
	hasIndent := !res.Context.IsPro
	return res.XML(res.data["xml"], hasIndent)
}

// ServeYAML sends yaml response.
func (res *asanaResponse) ServeYAML() error {
	return res.YAML(res.data["yaml"])
}

// ServeProtoBuf sends protobuf response.
func (res *asanaResponse) ServeProtoBuf() error {
	return res.ProtoBuf(res.data["protobuf"])
}

// ServeHTML sends html response.
func (res *asanaResponse) ServeHTML() error {
	switch res.data["html"].(type) {
	case string:
		val := res.data["html"].(string)
		return res.HTML(val)
	case []byte:
		val := res.data["html"].([]byte)
		return res.HTMLBlob(val)
	default:
		return errors.New("no data found")
	}
}

// ServeText sends text plain response.
func (res *asanaResponse) ServeText() error {
	var data interface{}
	if v, ok := res.data["txt"]; ok {
		data = v
	} else {
		for _, v := range res.data {
			data = v
			break
		}
	}

	switch data.(type) {
	case string:
		val := data.(string)
		return res.Text(val)
	case []byte:
		val := data.([]byte)
		return res.TextBlob(val)
	default:
		return errors.New("no data found")
	}
}

// ServeFormatted serve YAML, XML, JSON, ProtoBuffer, Html or Text, depending on the value of the Accept header
func (res *asanaResponse) ServeFormatted(encoding ...bool) error {
	if res.Context.AcceptsJSON() {
		return res.ServeJSON(encoding...)
	} else if res.Context.AcceptsYAML() {
		return res.ServeYAML()
	} else if res.Context.AcceptsXML() {
		return res.ServeXML()
	} else if res.Context.AcceptsProtoBuf() {
		return res.ServeProtoBuf()
	} else if res.Context.AcceptsJSONP() {
		return res.ServeJSONP()
	} else if res.Context.AcceptsHTML() {
		return res.ServeHTML()
	}
	return res.ServeText()
}

// Reset init asanaResponse
func (res *asanaResponse) Reset(ctx *Context) Response {
	res.Context = ctx
	res.status = 0
	res.data = make(map[interface{}]interface{})
	res._xsrfToken = ""
	return res
}

// GetFlash set the data
func (res *asanaResponse) GetData() map[interface{}]interface{} {
	return res.data
}

// SetData set the data
func (res *asanaResponse) SetData(data map[interface{}]interface{}) Response {
	res.data = data
	return res
}

// SetFlash set the data depending on the accepted
func (res *asanaResponse) SetBody(data interface{}) Response {
	if res.Context.Accepts(ApplicationYAML) {
		res.PutData("yaml", data)
	} else if res.Context.Accepts(ApplicationXML, TextXML) {
		res.PutData("xml", data)
	} else if res.Context.Accepts(ApplicationProtoBuf) {
		res.PutData("protobuf", data)
	} else if res.Context.Accepts(ApplicationJSONP) {
		res.PutData("jsonp", data)
	} else if res.Context.Accepts(ApplicationJSON) {
		res.PutData("json", data)
	} else if res.Context.Accepts(TextHTML) {
		res.PutData("html", data)
	} else {
		res.PutData("txt", data)
	}
	return res
}

// PutData set the data depending on the accepted
func (res *asanaResponse) PutData(key interface{}, data interface{}) Response {
	res.data[key] = data
	return res
}

// Text writes plain text to.Body.
func (res *asanaResponse) Text(data string) error {
	return res.TextBlob([]byte(data))
}

// TextBlob writes plain text to.Body from []byte.
func (res *asanaResponse) TextBlob(data []byte) (err error) {
	_, err = res.ResponseWriter.Write(data)
	return
}

// HTML writes html text to.Body.
func (res *asanaResponse) HTML(html string) error {
	return res.HTMLBlob([]byte(html))
}

// HTMLBlob writes html text to.Body from []byte.
func (res *asanaResponse) HTMLBlob(data []byte) error {
	return res.Blob(TextHTML, data)
}

// Blob writes []byte to.Body.
func (res *asanaResponse) Blob(contentType string, b []byte) error {
	return res.SetHeader(HeaderContentType, getContentTypeHead(contentType)).Body(b)
}

// Header sets.Header item string via given key.
func (res *asanaResponse) SetHeader(key, val string) Response {
	res.ResponseWriter.Header().Set(key, val)
	return res
}

// Stream writes stream to.Body.
func (res *asanaResponse) Stream(contentType string, r io.Reader) (err error) {
	res.SetHeader(HeaderContentType, getContentTypeHead(contentType))
	_, err = io.Copy(res.ResponseWriter, r)
	return
}

// NoContent white res
func (res *asanaResponse) NoContent() error {
	res.ResponseWriter.WriteHeader(res.GetStatus())
	return nil
}

// Redirect header location redirect
func (res *asanaResponse) Redirect(url string) error {
	if !res.IsRedirect() {
		return errors.New("invalid redirect status code")
	}

	res.ResponseWriter.Header().Set(HeaderLocation, url)
	res.ResponseWriter.WriteHeader(res.GetStatus())
	return nil
}

// JSON writes json to.Bodyres.
// if encoding is true, it converts utf-8 to \u0000 type.
func (res *asanaResponse) JSON(data interface{}, hasIndent bool, encoding bool) error {
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(res.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	if encoding {
		content = []byte(stringsToJSON(string(content)))
	}
	return res.SetHeader(HeaderContentType, getContentTypeHead(ApplicationJSON)).Body(content)
}

// ProtoBuf writes protobuf to.Body.
func (res *asanaResponse) ProtoBuf(data interface{}) error {
	content, err := proto.Encode(data)
	if err != nil {
		http.Error(res.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return res.SetHeader(HeaderContentType, getContentTypeHead(ApplicationProtoBuf)).Body(content)
}

// YAML writes yaml to.Body.
func (res *asanaResponse) YAML(data interface{}) error {
	content, err := yaml.Encode(data)
	if err != nil {
		http.Error(res.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return res.SetHeader(HeaderContentType, getContentTypeHead(ApplicationYAML)).Body(content)
}

// JSONP writes jsonp to.Body.
func (res *asanaResponse) JSONP(data interface{}, hasIndent bool) error {
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(res.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := res.Context.Request().Query("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback = template.JSEscapeString(callback)
	callbackContent := bytes.NewBufferString(" if(window." + callback + ")" + callback)
	callbackContent.WriteString("(")
	callbackContent.Write(content)
	callbackContent.WriteString(");\r\n")
	return res.SetHeader(HeaderContentType, getContentTypeHead(ApplicationJSONP)).Body(callbackContent.Bytes())
}

// XML writes xml string to.Body.
func (res *asanaResponse) XML(data interface{}, hasIndent bool) error {
	content, err := xml.Encode(data, hasIndent)
	if err != nil {
		http.Error(res.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return res.SetHeader(HeaderContentType, getContentTypeHead(ApplicationXML)).Body(content)
}

// Cookie sets cookie value via given key.
// others are ordered as cookie's max age time, path,domain, secure and httponly.
func (res *asanaResponse) SetCookie(name string, value string, others ...interface{}) Response {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "%s=%s", sanitizeName(name), sanitizeValue(value))

	//fix cookie not work in IE
	if len(others) > 0 {
		var maxAge int64

		switch v := others[0].(type) {
		case int:
			maxAge = int64(v)
		case int32:
			maxAge = int64(v)
		case int64:
			maxAge = v
		}

		switch {
		case maxAge > 0:
			_, _ = fmt.Fprintf(&b, "; Expires=%s; Max-Age=%d", time.Now().Add(time.Duration(maxAge) * time.Second).UTC().Format(time.RFC1123), maxAge)
		case maxAge < 0:
			_, _ = fmt.Fprintf(&b, "; Max-Age=0")
		}
	}

	// the settings below
	// Path, Domain, Secure, HttpOnly
	// can use nil skip set

	// default "/"
	if len(others) > 1 {
		if v, ok := others[1].(string); ok && len(v) > 0 {
			_, _ = fmt.Fprintf(&b, "; Path=%s", sanitizeValue(v))
		}
	} else {
		_, _ = fmt.Fprintf(&b, "; Path=%s", "/")
	}

	// default empty
	if len(others) > 2 {
		if v, ok := others[2].(string); ok && len(v) > 0 {
			_, _ = fmt.Fprintf(&b, "; Domain=%s", sanitizeValue(v))
		}
	}

	// default empty
	if len(others) > 3 {
		var secure bool
		switch v := others[3].(type) {
		case bool:
			secure = v
		default:
			if others[3] != nil {
				secure = true
			}
		}
		if secure {
			_, _ = fmt.Fprintf(&b, "; Secure")
		}
	}

	// default false. for session cookie default true
	if len(others) > 4 {
		if v, ok := others[4].(bool); ok && v {
			_, _ = fmt.Fprintf(&b, "; HttpOnly")
		}
	}

	res.ResponseWriter.Header().Add(HeaderSetCookie, b.String())

	return res
}

var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")

func sanitizeName(n string) string {
	return cookieNameSanitizer.Replace(n)
}

var cookieValueSanitizer = strings.NewReplacer("\n", " ", "\r", " ", ";", " ")

func sanitizeValue(v string) string {
	return cookieValueSanitizer.Replace(v)
}

func jsonRenderer(value interface{}) Renderer {
	return rendererFunc(func(ctx *Context) {
		_ = ctx.JSON(value, false, false)
	})
}

func errorRenderer(err error) Renderer {
	return rendererFunc(func(ctx *Context) {
		_ = ctx.SetStatus(500).Body([]byte(err.Error()))
	})
}

// SetXSRFToken creates a xsrf token string and returns.
func (res *asanaResponse) SetXSRFToken(key string, expire int64) string {
	if res._xsrfToken == "" {
		token, ok := res.GetSecureCookie(key, "_xsrf")
		if !ok {
			token = string(utils.RandomCreateBytes(32))
			res.SetSecureCookie(key, "_xsrf", token, expire)
		}
		res._xsrfToken = token
	}
	return res._xsrfToken
}

// GetSecureCookie Get secure cookie from request by a given key.
func (res *asanaResponse) GetSecureCookie(Secret, key string) (string, bool) {
	val := res.Context.Cookie(key)
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
	result, _ := base64.URLEncoding.DecodeString(vs)
	return string(result), true
}

// SetSecureCookie Set Secure cookie for response.
func (res *asanaResponse) SetSecureCookie(Secret, name, value string, others ...interface{}) Response {
	vs := base64.URLEncoding.EncodeToString([]byte(value))
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	h := hmac.New(sha1.New, []byte(Secret))
	_, _ = fmt.Fprintf(h, "%s%s", vs, timestamp)
	sig := fmt.Sprintf("%02x", h.Sum(nil))
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	res.SetCookie(name, cookie, others...)
	return res
}

// XSRFToken get _xsrfToken value
func (res *asanaResponse) XSRFToken() string {
	return res._xsrfToken
}

// RenderMethodResult renders the return value of a controller method to the output
func (res *asanaResponse) RenderMethodResult(result interface{}) Response {
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
		renderer.Render(res.Context)
	}
	return res
}

// Body sets.Body content.
// if EnableGzip, compress content string.
// it sends out.Body directly.
func (res *asanaResponse) Body(content []byte) error {
	var encoding string
	var buf = &bytes.Buffer{}
	if res.EnableGzip || res.Context.IsPro {
		encoding = ParseEncoding(res.Context.HTTPRequest)
	}
	if b, n, _ := WriteBody(encoding, buf, content); b {
		res.SetHeader(HeaderContentEncoding, n)
		res.SetHeader(HeaderContentLength, strconv.Itoa(buf.Len()))
	} else {
		res.SetHeader(HeaderContentLength, strconv.Itoa(len(content)))
	}
	// Write status code if it has been set manually
	// Set it to 0 afterwards to prevent "multiple response.WriteHeader calls"
	if res.GetStatus() != 0 {
		res.ResponseWriter.WriteHeader(res.GetStatus())
		res.SetStatus(0)
	} else {
		res.ResponseWriter.Started = true
	}
	_, _ = io.Copy(res.ResponseWriter, buf)
	return nil
}

// Download forces response for download file.
// it prepares the download.Header automatically.
func (res *asanaResponse) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(res.ResponseWriter, res.Context.HTTPRequest, file)
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
	res.SetHeader(HeaderContentDisposition, "attachment; "+fn)
	res.SetHeader(HeaderContentDescription, "File Transfer")
	res.SetHeader(HeaderContentType, "application/octet-stream")
	res.SetHeader(HeaderContentTransferEncoding, "binary")
	res.SetHeader(HeaderExpires, "0")
	res.SetHeader(HeaderCacheControl, "must-revalidate")
	res.SetHeader(HeaderPragma, "public")
	http.ServeFile(res.ResponseWriter, res.Context.HTTPRequest, file)
}

func getContentTypeHead(contentType string) string {
	return fmt.Sprintf("%s; charset=utf-8", contentType)
}

// GetStatus obtain the current status code
func (res *asanaResponse) GetStatus() int {
	return res.status
}

// Setstatus sets res status code.
// It writes.Header directly.
func (res *asanaResponse) SetStatus(status int) Response {
	res.status = status
	return res
}

// IsCachable returns boolean of this request is cached.
// HTTP 304 means cached.
func (res *asanaResponse) IsCachable() bool {
	return res.status >= 200 && res.status < 300 || res.status == 304
}

// IsEmpty returns boolean of this request is empty.
// HTTP 201，204 and 304 means empty.
func (res *asanaResponse) IsEmpty() bool {
	return res.status == 201 || res.status == 204 || res.status == 304
}

// IsOk returns boolean of this request runs well.
// HTTP 200 means ok.
func (res *asanaResponse) IsOk() bool {
	return res.status == 200
}

// IsSuccessful returns boolean of this request runs successfully.
// HTTP 2xx means ok.
func (res *asanaResponse) IsSuccessful() bool {
	return res.status >= 200 && res.status < 300
}

// IsRedirect returns boolean of this request is redirection header.
// HTTP 301,302,307 means redirection.
func (res *asanaResponse) IsRedirect() bool {
	return res.status == 301 || res.status == 302 || res.status == 303 || res.status == 307
}

// IsForbidden returns boolean of this request is forbidden.
// HTTP 403 means forbidden.
func (res *asanaResponse) IsForbidden() bool {
	return res.status == 403
}

// IsNotFound returns boolean of this request is not found.
// HTTP 404 means not found.
func (res *asanaResponse) IsNotFound() bool {
	return res.status == 404
}

// IsClientError returns boolean of this request client sends error data.
// HTTP 4xx means client error.
func (res *asanaResponse) IsClientError() bool {
	return res.status >= 400 && res.status < 500
}

// IsServerError returns boolean of this server handler errors.
// HTTP 5xx means server internal error.
func (res *asanaResponse) IsServerError() bool {
	return res.status >= 500 && res.status < 600
}

func stringsToJSON(str string) string {
	var jsons bytes.Buffer
	for _, r := range str {
		rint := int(r)
		if rint < 128 {
			jsons.WriteRune(r)
		} else {
			jsons.WriteString("\\u")
			if rint < 0x100 {
				jsons.WriteString("00")
			} else if rint < 0x1000 {
				jsons.WriteString("0")
			}
			jsons.WriteString(strconv.FormatInt(int64(rint), 16))
		}
	}
	return jsons.String()
}

// Abort stops this request.
// if asana.ErrorMaps exists, panic body.
func (res *asanaResponse) Abort(body string) error {
	panic(body)
	return nil
}

//ResponseWriter is a wrapper for the http.ResponseWriter
//started set to true if response was written to then don't execute other handler
type ResponseWriter struct {
	http.ResponseWriter
	Started bool
	Status  int
	Elapsed time.Duration
}

func (r *ResponseWriter) reset(rw http.ResponseWriter) {
	r.ResponseWriter = rw
	r.Status = 0
	r.Started = false
}

// Write writes the data to the connection as part of an HTTP reply,
// and sets `started` to true.
// started means the response has sent out.
func (r *ResponseWriter) Write(p []byte) (int, error) {
	r.Started = true
	return r.ResponseWriter.Write(p)
}

// WriteHeader sends an HTTP.Header with status code,
// and sets `started` to true.
func (r *ResponseWriter) WriteHeader(code int) {
	if r.Status > 0 {
		//prevent multiple response.WriteHeader calls
		return
	}
	r.Status = code
	r.Started = true
	r.ResponseWriter.WriteHeader(code)
}

// Hijack hijacker for http
func (r *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("webserver doesn't support hijacking")
	}
	return hj.Hijack()
}

// Flush http.Flusher
func (r *ResponseWriter) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Pusher http.Pusher
func (r *ResponseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
