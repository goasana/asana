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
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goasana/config/encoder/json"
	"github.com/goasana/config/encoder/proto"
	"github.com/goasana/config/encoder/xml"
	"github.com/goasana/config/encoder/yaml"
)

// AsanaResponse does work for sending response header.
type AsanaResponse struct {
	Context    *Context
	Status     int
	EnableGzip bool
}

// NewResponse returns new AsanaResponse.
// it contains nothing now.
func NewResponse() *AsanaResponse {
	return &AsanaResponse{}
}

// Reset init AsanaResponse
func (response *AsanaResponse) Reset(ctx *Context) {
	response.Context = ctx
	response.Status = 0
}

// Header sets response header item string via given key.
func (response *AsanaResponse) Header(key, val string) {
	response.Context.ResponseWriter.Header().Set(key, val)
}

// Body sets response body content.
// if EnableGzip, compress content string.
// it sends out response body directly.
func (response *AsanaResponse) Body(content []byte) error {
	var encoding string
	var buf = &bytes.Buffer{}
	if response.EnableGzip {
		encoding = ParseEncoding(response.Context.Request)
	}
	if b, n, _ := WriteBody(encoding, buf, content); b {
		response.Header("Content-Encoding", n)
		response.Header("Content-Length", strconv.Itoa(buf.Len()))
	} else {
		response.Header("Content-Length", strconv.Itoa(len(content)))
	}
	// Write status code if it has been set manually
	// Set it to 0 afterwards to prevent "multiple response.WriteHeader calls"
	if response.Status != 0 {
		response.Context.ResponseWriter.WriteHeader(response.Status)
		response.Status = 0
	} else {
		response.Context.ResponseWriter.Started = true
	}
	_, _ = io.Copy(response.Context.ResponseWriter, buf)
	return nil
}

// Cookie sets cookie value via given key.
// others are ordered as cookie's max age time, path,domain, secure and httponly.
func (response *AsanaResponse) Cookie(name string, value string, others ...interface{}) {
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

	response.Context.ResponseWriter.Header().Add("Set-Cookie", b.String())
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
		_ = ctx.Output.JSON(value, false, false)
	})
}

func errorRenderer(err error) Renderer {
	return rendererFunc(func(ctx *Context) {
		ctx.Output.SetStatus(500)
		_ = ctx.Output.Body([]byte(err.Error()))
	})
}

func getContentTypeHead(contentType string) string {
	return fmt.Sprintf("%s; charset=utf-8", contentType)
}

// JSON writes json to response body.
// if encoding is true, it converts utf-8 to \u0000 type.
func (response *AsanaResponse) JSON(data interface{}, hasIndent bool, encoding bool) error {
	response.Header("Content-Type", getContentTypeHead(ApplicationJSON))
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(response.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	if encoding {
		content = []byte(stringsToJSON(string(content)))
	}
	return response.Body(content)
}

// ProtoBuf writes protobuf to response body.
func (response *AsanaResponse) ProtoBuf(data interface{}) error {
	response.Header("Content-Type", getContentTypeHead(ApplicationProtoBuf))
	content, err := proto.Encode(data)
	if err != nil {
		http.Error(response.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return response.Body(content)
}

// YAML writes yaml to response body.
func (response *AsanaResponse) YAML(data interface{}) error {
	response.Header("Content-Type", getContentTypeHead(ApplicationYAML))
	content, err := yaml.Encode(data)
	if err != nil {
		http.Error(response.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return response.Body(content)
}

// JSONP writes jsonp to response body.
func (response *AsanaResponse) JSONP(data interface{}, hasIndent bool) error {
	response.Header("Content-Type", getContentTypeHead(ApplicationJSONP))
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(response.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := response.Context.Input.Query("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback = template.JSEscapeString(callback)
	callbackContent := bytes.NewBufferString(" if(window." + callback + ")" + callback)
	callbackContent.WriteString("(")
	callbackContent.Write(content)
	callbackContent.WriteString(");\r\n")
	return response.Body(callbackContent.Bytes())
}

// XML writes xml string to response body.
func (response *AsanaResponse) XML(data interface{}, hasIndent bool) error {
	response.Header("Content-Type", getContentTypeHead(ApplicationXML))
	content, err := xml.Encode(data, hasIndent)
	if err != nil {
		http.Error(response.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return response.Body(content)
}

// ServeFormatted serve YAML, XML OR JSON, depending on the value of the Accept header
func (response *AsanaResponse) ServeFormatted(data interface{}, hasIndent bool, hasEncode ...bool) {
	accept := response.Context.Input.Header("Accept")
	switch accept {
	case ApplicationYAML:
		_ = response.YAML(data)
	case ApplicationXML, TextXML:
		_ = response.XML(data, hasIndent)
	case ApplicationProtoBuf:
		_ = response.ProtoBuf(data)
	case ApplicationJSONP:
		_ = response.JSONP(data, hasIndent)
	default:
		_ = response.JSON(data, hasIndent, len(hasEncode) > 0 && hasEncode[0])
	}
}

// Download forces response for download file.
// it prepares the download response header automatically.
func (response *AsanaResponse) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(response.Context.ResponseWriter, response.Context.Request, file)
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
	response.Header("Content-Disposition", "attachment; "+fn)
	response.Header("Content-Description", "File Transfer")
	response.Header("Content-Type", "application/octet-stream")
	response.Header("Content-Transfer-Encoding", "binary")
	response.Header("Expires", "0")
	response.Header("Cache-Control", "must-revalidate")
	response.Header("Pragma", "public")
	http.ServeFile(response.Context.ResponseWriter, response.Context.Request, file)
}

// ContentType sets the content type from ext string.
// MIME type is given in mime package.
func (response *AsanaResponse) ContentType(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		response.Header("Content-Type", ctype)
	}
}

// SetStatus sets response status code.
// It writes response header directly.
func (response *AsanaResponse) SetStatus(status int) {
	response.Status = status
}

// IsCachable returns boolean of this request is cached.
// HTTP 304 means cached.
func (response *AsanaResponse) IsCachable() bool {
	return response.Status >= 200 && response.Status < 300 || response.Status == 304
}

// IsEmpty returns boolean of this request is empty.
// HTTP 201，204 and 304 means empty.
func (response *AsanaResponse) IsEmpty() bool {
	return response.Status == 201 || response.Status == 204 || response.Status == 304
}

// IsOk returns boolean of this request runs well.
// HTTP 200 means ok.
func (response *AsanaResponse) IsOk() bool {
	return response.Status == 200
}

// IsSuccessful returns boolean of this request runs successfully.
// HTTP 2xx means ok.
func (response *AsanaResponse) IsSuccessful() bool {
	return response.Status >= 200 && response.Status < 300
}

// IsRedirect returns boolean of this request is redirection header.
// HTTP 301,302,307 means redirection.
func (response *AsanaResponse) IsRedirect() bool {
	return response.Status == 301 || response.Status == 302 || response.Status == 303 || response.Status == 307
}

// IsForbidden returns boolean of this request is forbidden.
// HTTP 403 means forbidden.
func (response *AsanaResponse) IsForbidden() bool {
	return response.Status == 403
}

// IsNotFound returns boolean of this request is not found.
// HTTP 404 means not found.
func (response *AsanaResponse) IsNotFound() bool {
	return response.Status == 404
}

// IsClientError returns boolean of this request client sends error data.
// HTTP 4xx means client error.
func (response *AsanaResponse) IsClientError() bool {
	return response.Status >= 400 && response.Status < 500
}

// IsServerError returns boolean of this server handler errors.
// HTTP 5xx means server internal error.
func (response *AsanaResponse) IsServerError() bool {
	return response.Status >= 500 && response.Status < 600
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

// Session sets session item value with given key.
func (response *AsanaResponse) Session(name interface{}, value interface{}) {
	_ = response.Context.Input.CruSession.Set(name, value)
}
