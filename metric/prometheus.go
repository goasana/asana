// Copyright 2020 astaxie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/goasana/asana"
	"github.com/goasana/asana/logs"
)

func PrometheusMiddleWare(next http.Handler) http.Handler {
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      "asana",
		Subsystem: "http_request",
		ConstLabels: map[string]string{
			"server":  asana.BConfig.ServerName,
			"env":     asana.BConfig.RunMode,
			"appname": asana.BConfig.AppName,
		},
		Help: "The statics info for http request",
	}, []string{"pattern", "method", "status", "duration"})

	prometheus.MustRegister(summaryVec)

	registerBuildInfo()

	return http.HandlerFunc(func(writer http.ResponseWriter, q *http.Request) {
		start := time.Now()
		next.ServeHTTP(writer, q)
		end := time.Now()
		go report(end.Sub(start), writer, q, summaryVec)
	})
}

func registerBuildInfo() {
	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "asana",
		Subsystem: "build_info",
		Help:      "The building information",
		ConstLabels: map[string]string{
			"appname":        asana.BConfig.AppName,
			"build_version":  asana.BuildVersion,
			"build_revision": asana.BuildGitRevision,
			"build_status":   asana.BuildStatus,
			"build_tag":      asana.BuildTag,
			"build_time":     strings.Replace(asana.BuildTime, "--", " ", 1),
			"go_version":     asana.GoVersion,
			"git_branch":     asana.GitBranch,
			"start_time":     time.Now().Format("2006-01-02 15:04:05"),
		},
	}, []string{})

	prometheus.MustRegister(buildInfo)
	buildInfo.WithLabelValues().Set(1)
}

func report(dur time.Duration, writer http.ResponseWriter, q *http.Request, vec *prometheus.SummaryVec) {
	ctrl := asana.AsanaApp.Handlers
	ctx := ctrl.GetContext()
	ctx.Reset(writer, q)
	defer ctrl.GiveBackContext(ctx)

	// We cannot read the status code from q.Response.StatusCode
	// since the http server does not set q.Response. So q.Response is nil
	// Thus, we use reflection to read the status from writer whose concrete type is http.response
	responseVal := reflect.ValueOf(writer).Elem()
	field := responseVal.FieldByName("status")
	status := -1
	if field.IsValid() && field.Kind() == reflect.Int {
		status = int(field.Int())
	}
	ptn := "UNKNOWN"
	if rt, found := ctrl.FindRouter(ctx); found {
		ptn = rt.GetPattern()
	} else {
		logs.Warn("we can not find the router info for this request, so request will be recorded as UNKNOWN: " + q.URL.String())
	}
	ms := dur / time.Millisecond
	vec.WithLabelValues(ptn, q.Method, strconv.Itoa(status), strconv.Itoa(int(ms))).Observe(float64(ms))
}
