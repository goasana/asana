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

// Package apiauth provides handlers to enable apiauth support.
//
// Simple Usage:
//	import(
//		"github.com/goasana/asana"
//		"github.com/goasana/asana/plugins/apiauth"
//	)
//
//	func main(){
//		// apiauth every request
//		asana.InsertFilter("*", asana.BeforeRouter, apiauth.APIBaiscAuth("appid","appkey"))
//		asana.Run()
//	}
//
// Advanced Usage:
//
//	func getAppSecret(appid string) string {
//		// get appsecret by appid
//		// maybe store in configure, maybe in database
//	}
//
//	asana.InsertFilter("*", asana.BeforeRouter,apiauth.APISecretAuth(getAppSecret, 360))
//
// Information:
//
// In the request user should include these params in the query
//
// 1. appid
//
//		 appid is assigned to the application
//
// 2. signature
//
//	get the signature use apiauth.Signature()
//
//	when you send to server remember use url.QueryEscape()
//
// 3. timestamp:
//
//       send the request time, the format is yyyy-mm-dd HH:ii:ss
//
package apiauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/goasana/asana"
	"github.com/goasana/asana/context"
)

// AppIDToAppSecret is used to get appsecret throw appid
type AppIDToAppSecret func(string) string

// APIBasicAuth use the basic appid/appkey as the AppIdToAppSecret
func APIBasicAuth(appID, appKey string) asana.FilterFunc {
	ft := func(aid string) string {
		if aid == appID {
			return appKey
		}
		return ""
	}
	return APISecretAuth(ft, 300)
}

// APIBaiscAuth calls APIBasicAuth for previous callers
func APIBaiscAuth(appID, appKey string) asana.FilterFunc {
	return APIBasicAuth(appID, appKey)
}

// APISecretAuth use AppIdToAppSecret verify and
func APISecretAuth(f AppIDToAppSecret, timeout int) asana.FilterFunc {
	return func(ctx *context.Context) {
		if ctx.Request.Query("appid") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: appid")
			return
		}
		appSecret := f(ctx.Request.Query("appid"))
		if appSecret == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("not exist this appid")
			return
		}
		if ctx.Request.Query("signature") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: signature")
			return
		}
		if ctx.Request.Query("timestamp") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("miss query param: timestamp")
			return
		}
		u, err := time.Parse("2006-01-02 15:04:05", ctx.Request.Query("timestamp"))
		if err != nil {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("timestamp format is error, should 2006-01-02 15:04:05")
			return
		}
		t := time.Now()
		if t.Sub(u).Seconds() > float64(timeout) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("timeout! the request time is long ago, please try again")
			return
		}
		if ctx.Request.Query("signature") !=
			Signature(appSecret, ctx.Request.Method(), ctx.HTTPRequest.Form, ctx.Request.URL()) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("auth failed")
		}
	}
}

// Signature used to generate signature with the appsecret/method/params/RequestURI
func Signature(appSecret, method string, params url.Values, RequestURL string) (result string) {
	var b bytes.Buffer
	keys := make([]string, len(params))
	pa := make(map[string]string)
	for k, v := range params {
		pa[k] = v[0]
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if key == "signature" {
			continue
		}

		val := pa[key]
		if key != "" && val != "" {
			b.WriteString(key)
			b.WriteString(val)
		}
	}

	stringToSign := fmt.Sprintf("%v\n%v\n%v\n", method, b.String(), RequestURL)

	sha256hash := sha256.New
	hash := hmac.New(sha256hash, []byte(appSecret))
	hash.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
