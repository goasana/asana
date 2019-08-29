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

// Package cors provides handlers to enable CORS support.
// Usage
//	import (
// 		"github.com/goasana/asana"
//		"github.com/goasana/asana/plugins/cors"
// )
//
//	func main() {
//		// CORS for https://foo.* origins, allowing:
//		// - PUT and PATCH methods
//		// - Origin header
//		// - Credentials share
//		asana.InsertFilter("*", asana.BeforeRouter, cors.Allow(&cors.Options{
//			AllowOrigins:     []string{"https://*.foo.com"},
//			AllowMethods:     []string{"PUT", "PATCH"},
//			AllowHeaders:     []string{"Origin"},
//			ExposeHeaders:    []string{"Content-Length"},
//			AllowCredentials: true,
//		}))
//		asana.Run()
//	}
package cors

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goasana/asana"
	"github.com/goasana/asana/context"
)

var (
	defaultAllowHeaders = []string{"Origin", "Accept", "Content-Type", "Authorization"}
	// Regex patterns are generated from AllowOrigins. These are used and generated internally.
	allowOriginPatterns []string
)

// Options represents Access Control options.
type Options struct {
	// If set, all origins are allowed.
	AllowAllOrigins bool
	// A list of allowed origins. Wild cards and FQDNs are supported.
	AllowOrigins []string
	// If set, allows to share auth credentials such as cookies.
	AllowCredentials bool
	// A list of allowed HTTP methods.
	AllowMethods []string
	// A list of allowed HTTP headers.
	AllowHeaders []string
	// A list of exposed HTTP headers.
	ExposeHeaders []string
	// Max age of the CORS headers.
	MaxAge time.Duration
}

// Header converts options into CORS headers.
func (o *Options) Header(origin string) (headers map[string]string) {
	headers = make(map[string]string)
	// if origin is not allowed, don't extend the headers
	// with CORS headers.
	if !o.AllowAllOrigins && !o.IsOriginAllowed(origin) {
		return
	}

	// add allow origin
	if o.AllowAllOrigins {
		headers[context.HeaderAccessControlAllowOrigin] = "*"
	} else {
		headers[context.HeaderAccessControlAllowOrigin] = origin
	}

	// add allow credentials
	headers[context.HeaderAccessControlAllowCredentials] = strconv.FormatBool(o.AllowCredentials)

	// add allow methods
	if len(o.AllowMethods) > 0 {
		headers[context.HeaderAccessControlAllowMethods] = strings.Join(o.AllowMethods, ",")
	}

	// add allow headers
	if len(o.AllowHeaders) > 0 {
		headers[context.HeaderAccessControlAllowHeaders] = strings.Join(o.AllowHeaders, ",")
	}

	// add exposed header
	if len(o.ExposeHeaders) > 0 {
		headers[context.HeaderAccessControlExposeHeaders] = strings.Join(o.ExposeHeaders, ",")
	}
	// add a max age header
	if o.MaxAge > time.Duration(0) {
		headers[context.HeaderAccessControlMaxAge] = strconv.FormatInt(int64(o.MaxAge/time.Second), 10)
	}
	return
}

// PreflightHeader converts options into CORS headers for a preflight response.
func (o *Options) PreflightHeader(origin, rMethod, rHeaders string) (headers map[string]string) {
	headers = make(map[string]string)
	if !o.AllowAllOrigins && !o.IsOriginAllowed(origin) {
		return
	}
	// verify if requested method is allowed
	for _, method := range o.AllowMethods {
		if method == rMethod {
			headers[context.HeaderAccessControlAllowMethods] = strings.Join(o.AllowMethods, ",")
			break
		}
	}

	// verify if requested headers are allowed
	var allowed []string
	for _, rHeader := range strings.Split(rHeaders, ",") {
		rHeader = strings.TrimSpace(rHeader)
	lookupLoop:
		for _, allowedHeader := range o.AllowHeaders {
			if strings.EqualFold(rHeader, allowedHeader) {
				allowed = append(allowed, rHeader)
				break lookupLoop
			}
		}
	}

	headers[context.HeaderAccessControlAllowCredentials] = strconv.FormatBool(o.AllowCredentials)
	// add allow origin
	if o.AllowAllOrigins {
		headers[context.HeaderAccessControlAllowOrigin] = "*"
	} else {
		headers[context.HeaderAccessControlAllowOrigin] = origin
	}

	// add allowed headers
	if len(allowed) > 0 {
		headers[context.HeaderAccessControlAllowHeaders] = strings.Join(allowed, ",")
	}

	// add exposed headers
	if len(o.ExposeHeaders) > 0 {
		headers[context.HeaderAccessControlExposeHeaders] = strings.Join(o.ExposeHeaders, ",")
	}
	// add a max age header
	if o.MaxAge > time.Duration(0) {
		headers[context.HeaderAccessControlMaxAge] = strconv.FormatInt(int64(o.MaxAge/time.Second), 10)
	}
	return
}

// IsOriginAllowed looks up if the origin matches one of the patterns
// generated from Options.AllowOrigins patterns.
func (o *Options) IsOriginAllowed(origin string) (allowed bool) {
	for _, pattern := range allowOriginPatterns {
		allowed, _ = regexp.MatchString(pattern, origin)
		if allowed {
			return
		}
	}
	return
}

// Allow enables CORS for requests those match the provided options.
func Allow(opts *Options) asana.FilterFunc {
	// Allow default headers if nothing is specified.
	if len(opts.AllowHeaders) == 0 {
		opts.AllowHeaders = defaultAllowHeaders
	}

	for _, origin := range opts.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.Replace(pattern, "\\*", ".*", -1)
		pattern = strings.Replace(pattern, "\\?", ".", -1)
		allowOriginPatterns = append(allowOriginPatterns, "^"+pattern+"$")
	}

	return func(ctx *context.Context) {
		var (
			origin           = ctx.Header(context.HeaderOrigin)
			requestedMethod  = ctx.Header(context.HeaderAccessControlRequestMethod)
			requestedHeaders = ctx.Header(context.HeaderAccessControlRequestHeaders)
			// additional headers to be added
			// to the response.
			headers map[string]string
		)

		headers = opts.Header(origin)

		if ctx.Method() == "OPTIONS" &&
			(requestedMethod != "" || requestedHeaders != "") {
			headers = opts.PreflightHeader(origin, requestedMethod, requestedHeaders)
			for key, value := range headers {
				ctx.SetHeader(key, value)
			}
			ctx.ResponseWriter.WriteHeader(http.StatusOK)
			return
		}


		for key, value := range headers {
			ctx.SetHeader(key, value)
		}
	}
}
