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
	"fmt"
	"strconv"
	"strings"
	"time"
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

// AsanaResponse does work for sending.Header.
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

// Cookie sets cookie value via given key.
// others are ordered as cookie's max age time, path,domain, secure and httponly.
func (response *AsanaResponse) Cookie(name string, value string, others ...interface{}) *AsanaResponse {
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

	response.Context.ResponseWriter.Header().Add(HeaderSetCookie, b.String())

	return response
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
		ctx.Response.SetStatus(500)
		_ = ctx.Body([]byte(err.Error()))
	})
}

func getContentTypeHead(contentType string) string {
	return fmt.Sprintf("%s; charset=utf-8", contentType)
}


// SetStatus sets response status code.
// It writes.Header directly.
func (response *AsanaResponse) SetStatus(status int) *AsanaResponse {
	response.Status = status
	return response
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
func (response *AsanaResponse) Session(name interface{}, value interface{}) *AsanaResponse {
	_ = response.Context.Request.CruSession.Set(name, value)
	return response
}
