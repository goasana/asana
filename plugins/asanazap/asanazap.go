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

// Package cors provides handlers to enable ZAP log.
//package main
//
//import (
//	"time"
//
//	"go.uber.org/zap"
//
//	asana "github.com/goasana/framework"
//	"github.com/goasana/framework/plugins/asanazap"
//)
//
//type MainController struct {
//	asana.Controller
//}
//
//func (this *MainController) Get() {
//	this.Ctx.WriteString("hello world")
//}
//
//func main() {
//	logger, _ := zap.NewProduction()
//	asanazap.InitAsanaZapMiddleware(logger, time.RFC3339, true)
//	asana.Router("/", &MainController{})
//	asana.Run(":8090")
//}
package asanazap

import (
	"time"

	"go.uber.org/zap"

	"github.com/goasana/config/encoder/json"
	"github.com/goasana/framework"
	"github.com/goasana/framework/context"
	"github.com/goasana/framework/logs"
)

func BeforeMiddlewareZap() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		ctx.Input.SetData("start_timer", time.Now())
	}
}

func AfterMiddlewareZap(logger *zap.Logger, timeFormat string, utc bool, appendBody bool) func(ctx *context.Context) {
	if appendBody {
		logs.Warn("[asanazap] Be careful with personal data in body.")
	}

	return func(ctx *context.Context) {
		startTimeInterface := ctx.Input.GetData("start_timer")
		if startTime, ok := startTimeInterface.(time.Time); ok {
			path := ctx.Request.URL.Path
			query := ctx.Request.URL.RawQuery

			endTime := time.Now()
			latency := endTime.Sub(startTime)

			if utc {
				endTime = endTime.UTC()
			}

			headers, _ := json.Encode(ctx.Request.Header, false)

			statusCode := ctx.Output.Status

			// TODO: The default code in asana is 0.
			if statusCode == 0 {
				statusCode = 200
			}

			fields := []zap.Field{
				zap.Int("status", statusCode),
				zap.String("method", ctx.Input.Method()),
				zap.String("path", path),
				zap.String("uri", ctx.Input.URI()),
				zap.String("query", query),
				zap.ByteString("headers", headers),
				zap.String("site", ctx.Input.Site()),
				zap.String("ip", ctx.Input.IP()),
				zap.String("refer", ctx.Input.Refer()),
				zap.String("user-agent", ctx.Input.UserAgent()),
				zap.String("time", endTime.Format(timeFormat)),
				zap.Duration("latency", latency),
			}

			if appendBody {
				fields = append(fields, zap.ByteString("body", ctx.Input.RequestBody))
			}

			logger.Info(path, fields...)
		}
	}
}

func InitAsanaZapMiddleware(logger *zap.Logger, timeFormat string, utc bool, appendBody ...bool) {
	asana.InsertFilter("*", asana.BeforeRouter, BeforeMiddlewareZap(), false)
	asana.InsertFilter("*", asana.FinishRouter, AfterMiddlewareZap(logger, timeFormat, utc, len(appendBody) > 0 && appendBody[0]), false)

	logs.Info("[asanazap] Logger started")
}
