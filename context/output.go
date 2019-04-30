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

	"github.com/goasana/framework/encoder/json"
	"github.com/goasana/framework/encoder/proto"
	"github.com/goasana/framework/encoder/xml"
	"github.com/goasana/framework/encoder/yaml"
)

// AsanaOutput does work for sending response header.
type AsanaOutput struct {
	Context    *Context
	Status     int
	EnableGzip bool
}

// NewOutput returns new AsanaOutput.
// it contains nothing now.
func NewOutput() *AsanaOutput {
	return &AsanaOutput{}
}

// Reset init AsanaOutput
func (output *AsanaOutput) Reset(ctx *Context) {
	output.Context = ctx
	output.Status = 0
}

// Header sets response header item string via given key.
func (output *AsanaOutput) Header(key, val string) {
	output.Context.ResponseWriter.Header().Set(key, val)
}

// Body sets response body content.
// if EnableGzip, compress content string.
// it sends out response body directly.
func (output *AsanaOutput) Body(content []byte) error {
	var encoding string
	var buf = &bytes.Buffer{}
	if output.EnableGzip {
		encoding = ParseEncoding(output.Context.Request)
	}
	if b, n, _ := WriteBody(encoding, buf, content); b {
		output.Header("Content-Encoding", n)
		output.Header("Content-Length", strconv.Itoa(buf.Len()))
	} else {
		output.Header("Content-Length", strconv.Itoa(len(content)))
	}
	// Write status code if it has been set manually
	// Set it to 0 afterwards to prevent "multiple response.WriteHeader calls"
	if output.Status != 0 {
		output.Context.ResponseWriter.WriteHeader(output.Status)
		output.Status = 0
	} else {
		output.Context.ResponseWriter.Started = true
	}
	_, _ = io.Copy(output.Context.ResponseWriter, buf)
	return nil
}

// Cookie sets cookie value via given key.
// others are ordered as cookie's max age time, path,domain, secure and httponly.
func (output *AsanaOutput) Cookie(name string, value string, others ...interface{}) {
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

	output.Context.ResponseWriter.Header().Add("Set-Cookie", b.String())
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
func (output *AsanaOutput) JSON(data interface{}, hasIndent bool, encoding bool) error {
	output.Header("Content-Type", getContentTypeHead(ApplicationJSON))
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	if encoding {
		content = []byte(stringsToJSON(string(content)))
	}
	return output.Body(content)
}

// ProtoBuf writes protobuf to response body.
func (output *AsanaOutput) ProtoBuf(data interface{}) error {
	output.Header("Content-Type", getContentTypeHead(ApplicationProtoBuf))
	content, err := proto.Encode(data)
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return output.Body(content)
}

// YAML writes yaml to response body.
func (output *AsanaOutput) YAML(data interface{}) error {
	output.Header("Content-Type", getContentTypeHead(ApplicationYAML))
	content, err := yaml.Encode(data)
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return output.Body(content)
}

// JSONP writes jsonp to response body.
func (output *AsanaOutput) JSONP(data interface{}, hasIndent bool) error {
	output.Header("Content-Type", getContentTypeHead(ApplicationJSONP))
	content, err := json.Encode(data, hasIndent)
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := output.Context.Input.Query("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback = template.JSEscapeString(callback)
	callbackContent := bytes.NewBufferString(" if(window." + callback + ")" + callback)
	callbackContent.WriteString("(")
	callbackContent.Write(content)
	callbackContent.WriteString(");\r\n")
	return output.Body(callbackContent.Bytes())
}

// XML writes xml string to response body.
func (output *AsanaOutput) XML(data interface{}, hasIndent bool) error {
	output.Header("Content-Type", getContentTypeHead(ApplicationXML))
	content, err := xml.Encode(data, hasIndent)
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return output.Body(content)
}

// ServeFormatted serve YAML, XML OR JSON, depending on the value of the Accept header
func (output *AsanaOutput) ServeFormatted(data interface{}, hasIndent bool, hasEncode ...bool) {
	accept := output.Context.Input.Header("Accept")
	switch accept {
	case ApplicationYAML:
		_ = output.YAML(data)
	case ApplicationXML, TextXML:
		_ = output.XML(data, hasIndent)
	case ApplicationProtoBuf:
		_ = output.ProtoBuf(data)
	case ApplicationJSONP:
		_ = output.JSONP(data, hasIndent)
	default:
		_ = output.JSON(data, hasIndent, len(hasEncode) > 0 && hasEncode[0])
	}
}

// Download forces response for download file.
// it prepares the download response header automatically.
func (output *AsanaOutput) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(output.Context.ResponseWriter, output.Context.Request, file)
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
	output.Header("Content-Disposition", "attachment; "+fn)
	output.Header("Content-Description", "File Transfer")
	output.Header("Content-Type", "application/octet-stream")
	output.Header("Content-Transfer-Encoding", "binary")
	output.Header("Expires", "0")
	output.Header("Cache-Control", "must-revalidate")
	output.Header("Pragma", "public")
	http.ServeFile(output.Context.ResponseWriter, output.Context.Request, file)
}

// ContentType sets the content type from ext string.
// MIME type is given in mime package.
func (output *AsanaOutput) ContentType(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		output.Header("Content-Type", ctype)
	}
}

// SetStatus sets response status code.
// It writes response header directly.
func (output *AsanaOutput) SetStatus(status int) {
	output.Status = status
}

// IsCachable returns boolean of this request is cached.
// HTTP 304 means cached.
func (output *AsanaOutput) IsCachable() bool {
	return output.Status >= 200 && output.Status < 300 || output.Status == 304
}

// IsEmpty returns boolean of this request is empty.
// HTTP 201，204 and 304 means empty.
func (output *AsanaOutput) IsEmpty() bool {
	return output.Status == 201 || output.Status == 204 || output.Status == 304
}

// IsOk returns boolean of this request runs well.
// HTTP 200 means ok.
func (output *AsanaOutput) IsOk() bool {
	return output.Status == 200
}

// IsSuccessful returns boolean of this request runs successfully.
// HTTP 2xx means ok.
func (output *AsanaOutput) IsSuccessful() bool {
	return output.Status >= 200 && output.Status < 300
}

// IsRedirect returns boolean of this request is redirection header.
// HTTP 301,302,307 means redirection.
func (output *AsanaOutput) IsRedirect() bool {
	return output.Status == 301 || output.Status == 302 || output.Status == 303 || output.Status == 307
}

// IsForbidden returns boolean of this request is forbidden.
// HTTP 403 means forbidden.
func (output *AsanaOutput) IsForbidden() bool {
	return output.Status == 403
}

// IsNotFound returns boolean of this request is not found.
// HTTP 404 means not found.
func (output *AsanaOutput) IsNotFound() bool {
	return output.Status == 404
}

// IsClientError returns boolean of this request client sends error data.
// HTTP 4xx means client error.
func (output *AsanaOutput) IsClientError() bool {
	return output.Status >= 400 && output.Status < 500
}

// IsServerError returns boolean of this server handler errors.
// HTTP 5xx means server internal error.
func (output *AsanaOutput) IsServerError() bool {
	return output.Status >= 500 && output.Status < 600
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
func (output *AsanaOutput) Session(name interface{}, value interface{}) {
	_ = output.Context.Input.CruSession.Set(name, value)
}
