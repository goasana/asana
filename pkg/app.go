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

package asana

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"strings"
	"time"

	"github.com/goasana/asana/grace"
	"github.com/goasana/asana/logs"
	"github.com/goasana/asana/utils"
	"golang.org/x/crypto/acme/autocert"
)

var (
	// AsanaApp is an application instance
	AsanaApp *App
)

func init() {
	// create asana application
	AsanaApp = NewApp()
}

// App defines asana application with a new PatternServeMux.
type App struct {
	Handlers *ControllerRegister
	Server   *http.Server
}

// NewApp returns a new asana application.
func NewApp() *App {
	cr := NewControllerRegister()
	app := &App{Handlers: cr, Server: &http.Server{}}
	return app
}

// MiddleWare function for http.Handler
type MiddleWare func(http.Handler) http.Handler

// Run asana application.
func (app *App) Run(mws ...MiddleWare) {
	addr := BConfig.Listen.HTTPAddr

	if BConfig.Listen.HTTPPort != 0 {
		addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPAddr, BConfig.Listen.HTTPPort)
	}

	var (
		err        error
		l          net.Listener
		endRunning = make(chan bool, 1)
	)

	// run cgi server
	if BConfig.Listen.EnableFcgi {
		if BConfig.Listen.EnableStdIo {
			if err = fcgi.Serve(nil, app.Handlers); err == nil { // standard I/O
				logs.Info("Use FCGI via standard I/O")
			} else {
				logs.Critical("Cannot use FCGI via standard I/O", err)
			}
			return
		}
		if BConfig.Listen.HTTPPort == 0 {
			// remove the Socket file before start
			if utils.FileExists(addr) {
				os.Remove(addr)
			}
			l, err = net.Listen("unix", addr)
		} else {
			l, err = net.Listen("tcp", addr)
		}
		if err != nil {
			logs.Critical("Listen: ", err)
		}
		if err = fcgi.Serve(l, app.Handlers); err != nil {
			logs.Critical("fcgi.Serve: ", err)
		}
		return
	}

	app.Server.Handler = app.Handlers
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] == nil {
			continue
		}
		app.Server.Handler = mws[i](app.Server.Handler)
	}
	app.Server.ReadTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.WriteTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.ErrorLog = logs.GetLogger("HTTP")

	// run graceful mode
	if BConfig.Listen.Graceful {
		httpsAddr := BConfig.Listen.HTTPSAddr
		app.Server.Addr = httpsAddr
		if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
			go func() {
				time.Sleep(1000 * time.Microsecond)
				if BConfig.Listen.HTTPSPort != 0 {
					httpsAddr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
					app.Server.Addr = httpsAddr
				}
				server := grace.NewServer(httpsAddr, app.Server.Handler)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.EnableMutualHTTPS {
					if err := server.ListenAndServeMutualTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile, BConfig.Listen.TrustCaFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
					}
				} else {
					if BConfig.Listen.AutoTLS {
						m := autocert.Manager{
							Prompt:     autocert.AcceptTOS,
							HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
							Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
						}
						app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
						BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
					}
					if err := server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
					}
				}
				endRunning <- true
			}()
		}
		if BConfig.Listen.EnableHTTP {
			go func() {
				server := grace.NewServer(addr, app.Server.Handler)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.ListenTCP4 {
					server.Network = "tcp4"
				}
				if err := server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err, fmt.Sprintf("%d", os.Getpid()))
					time.Sleep(100 * time.Microsecond)
				}
				endRunning <- true
			}()
		}
		<-endRunning
		return
	}

	// run normal mode
	if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
		go func() {
			time.Sleep(1000 * time.Microsecond)
			if BConfig.Listen.HTTPSPort != 0 {
				app.Server.Addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
			} else if BConfig.Listen.EnableHTTP {
				logs.Info("Start https server error, conflict with http. Please reset https port")
				return
			}
			logs.Info("https server Running on https://%s", app.Server.Addr)
			if BConfig.Listen.AutoTLS {
				m := autocert.Manager{
					Prompt:     autocert.AcceptTOS,
					HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
					Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
				}
				app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
				BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
			} else if BConfig.Listen.EnableMutualHTTPS {
				pool := x509.NewCertPool()
				data, err := ioutil.ReadFile(BConfig.Listen.TrustCaFile)
				if err != nil {
					logs.Info("MutualHTTPS should provide TrustCaFile")
					return
				}
				pool.AppendCertsFromPEM(data)
				app.Server.TLSConfig = &tls.Config{
					ClientCAs:  pool,
					ClientAuth: tls.RequireAndVerifyClientCert,
				}
			}
			if err := app.Server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
				logs.Critical("ListenAndServeTLS: ", err)
				time.Sleep(100 * time.Microsecond)
				endRunning <- true
			}
		}()

	}
	if BConfig.Listen.EnableHTTP {
		go func() {
			app.Server.Addr = addr
			logs.Info("http server Running on http://%s", app.Server.Addr)
			if BConfig.Listen.ListenTCP4 {
				ln, err := net.Listen("tcp4", app.Server.Addr)
				if err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					return
				}
				if err = app.Server.Serve(ln); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					return
				}
			} else {
				if err := app.Server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}
		}()
	}
	<-endRunning
}

// Router adds a patterned controller handler to AsanaApp.
// it's an alias method of App.Router.
// usage:
//  simple router
//  asana.Router("/admin", &admin.UserController{})
//  asana.Router("/admin/index", &admin.ArticleController{})
//
//  regex router
//
//  asana.Router("/api/:id([0-9]+)", &controllers.RController{})
//
//  custom rules
//  asana.Router("/api/list",&RestController{},"*:ListFood")
//  asana.Router("/api/create",&RestController{},"post:CreateFood")
//  asana.Router("/api/update",&RestController{},"put:UpdateFood")
//  asana.Router("/api/delete",&RestController{},"delete:DeleteFood")
func Router(rootpath string, c ControllerInterface, mappingMethods ...string) *App {
	AsanaApp.Handlers.Add(rootpath, c, mappingMethods...)
	return AsanaApp
}

// UnregisterFixedRoute unregisters the route with the specified fixedRoute. It is particularly useful
// in web applications that inherit most routes from a base webapp via the underscore
// import, and aim to overwrite only certain paths.
// The method parameter can be empty or "*" for all HTTP methods, or a particular
// method type (e.g. "GET" or "POST") for selective removal.
//
// Usage (replace "GET" with "*" for all methods):
//  asana.UnregisterFixedRoute("/yourpreviouspath", "GET")
//  asana.Router("/yourpreviouspath", yourControllerAddress, "get:GetNewPage")
func UnregisterFixedRoute(fixedRoute string, method string) *App {
	subPaths := splitPath(fixedRoute)
	if method == "" || method == "*" {
		for m := range HTTPMETHOD {
			if _, ok := AsanaApp.Handlers.routers[m]; !ok {
				continue
			}
			if AsanaApp.Handlers.routers[m].prefix == strings.Trim(fixedRoute, "/ ") {
				findAndRemoveSingleTree(AsanaApp.Handlers.routers[m])
				continue
			}
			findAndRemoveTree(subPaths, AsanaApp.Handlers.routers[m], m)
		}
		return AsanaApp
	}
	// Single HTTP method
	um := strings.ToUpper(method)
	if _, ok := AsanaApp.Handlers.routers[um]; ok {
		if AsanaApp.Handlers.routers[um].prefix == strings.Trim(fixedRoute, "/ ") {
			findAndRemoveSingleTree(AsanaApp.Handlers.routers[um])
			return AsanaApp
		}
		findAndRemoveTree(subPaths, AsanaApp.Handlers.routers[um], um)
	}
	return AsanaApp
}

func findAndRemoveTree(paths []string, entryPointTree *Tree, method string) {
	for i := range entryPointTree.fixrouters {
		if entryPointTree.fixrouters[i].prefix == paths[0] {
			if len(paths) == 1 {
				if len(entryPointTree.fixrouters[i].fixrouters) > 0 {
					// If the route had children subtrees, remove just the functional leaf,
					// to allow children to function as before
					if len(entryPointTree.fixrouters[i].leaves) > 0 {
						entryPointTree.fixrouters[i].leaves[0] = nil
						entryPointTree.fixrouters[i].leaves = entryPointTree.fixrouters[i].leaves[1:]
					}
				} else {
					// Remove the *Tree from the fixrouters slice
					entryPointTree.fixrouters[i] = nil

					if i == len(entryPointTree.fixrouters)-1 {
						entryPointTree.fixrouters = entryPointTree.fixrouters[:i]
					} else {
						entryPointTree.fixrouters = append(entryPointTree.fixrouters[:i], entryPointTree.fixrouters[i+1:len(entryPointTree.fixrouters)]...)
					}
				}
				return
			}
			findAndRemoveTree(paths[1:], entryPointTree.fixrouters[i], method)
		}
	}
}

func findAndRemoveSingleTree(entryPointTree *Tree) {
	if entryPointTree == nil {
		return
	}
	if len(entryPointTree.fixrouters) > 0 {
		// If the route had children subtrees, remove just the functional leaf,
		// to allow children to function as before
		if len(entryPointTree.leaves) > 0 {
			entryPointTree.leaves[0] = nil
			entryPointTree.leaves = entryPointTree.leaves[1:]
		}
	}
}

// Include will generate router file in the router/xxx.go from the controller's comments
// usage:
// asana.Include(&BankAccount{}, &OrderController{},&RefundController{},&ReceiptController{})
// type BankAccount struct{
//   asana.Controller
// }
//
// register the function
// func (b *BankAccount)Mapping(){
//  b.Mapping("ShowAccount" , b.ShowAccount)
//  b.Mapping("ModifyAccount", b.ModifyAccount)
//}
//
// //@router /account/:id  [get]
// func (b *BankAccount) ShowAccount(){
//    //logic
// }
//
//
// //@router /account/:id  [post]
// func (b *BankAccount) ModifyAccount(){
//    //logic
// }
//
// the comments @router url methodlist
// url support all the function Router's pattern
// methodlist [get post head put delete options *]
func Include(cList ...ControllerInterface) *App {
	AsanaApp.Handlers.Include(cList...)
	return AsanaApp
}

// RESTRouter adds a restful controller handler to AsanaApp.
// its' controller implements asana.ControllerInterface and
// defines a param "pattern/:objectId" to visit each resource.
func RESTRouter(rootpath string, c ControllerInterface) *App {
	Router(rootpath, c)
	Router(path.Join(rootpath, ":objectId"), c)
	return AsanaApp
}

// AutoRouter adds defined controller handler to AsanaApp.
// it's same to App.AutoRouter.
// if asana.AddAuto(&MainContorlller{}) and MainController has methods List and Page,
// visit the url /main/list to exec List function or /main/page to exec Page function.
func AutoRouter(c ControllerInterface) *App {
	AsanaApp.Handlers.AddAuto(c)
	return AsanaApp
}

// AutoPrefix adds controller handler to AsanaApp with prefix.
// it's same to App.AutoRouterWithPrefix.
// if asana.AutoPrefix("/admin",&MainContorlller{}) and MainController has methods List and Page,
// visit the url /admin/main/list to exec List function or /admin/main/page to exec Page function.
func AutoPrefix(prefix string, c ControllerInterface) *App {
	AsanaApp.Handlers.AddAutoPrefix(prefix, c)
	return AsanaApp
}

// Get used to register router for Get method
// usage:
//    asana.Get("/", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Get(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Get(rootpath, f)
	return AsanaApp
}

// Post used to register router for Post method
// usage:
//    asana.Post("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Post(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Post(rootpath, f)
	return AsanaApp
}

// Delete used to register router for Delete method
// usage:
//    asana.Delete("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Delete(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Delete(rootpath, f)
	return AsanaApp
}

// Put used to register router for Put method
// usage:
//    asana.Put("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Put(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Put(rootpath, f)
	return AsanaApp
}

// Head used to register router for Head method
// usage:
//    asana.Head("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Head(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Head(rootpath, f)
	return AsanaApp
}

// Options used to register router for Options method
// usage:
//    asana.Options("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Options(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Options(rootpath, f)
	return AsanaApp
}

// Patch used to register router for Patch method
// usage:
//    asana.Patch("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Patch(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Patch(rootpath, f)
	return AsanaApp
}

// Any used to register router for all methods
// usage:
//    asana.Any("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func Any(rootpath string, f FilterFunc) *App {
	AsanaApp.Handlers.Any(rootpath, f)
	return AsanaApp
}

// Handler used to register a Handler router
// usage:
//    asana.Handler("/api", http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
//          fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
//    }))
func Handler(rootpath string, h http.Handler, options ...interface{}) *App {
	AsanaApp.Handlers.Handler(rootpath, h, options...)
	return AsanaApp
}

// InsertFilter adds a FilterFunc with pattern condition and action constant.
// The pos means action constant including
// asana.BeforeStatic, asana.BeforeRouter, asana.BeforeExec, asana.AfterExec and asana.FinishRouter.
// The bool params is for setting the returnOnOutput value (false allows multiple filters to execute)
func InsertFilter(pattern string, pos int, filter FilterFunc, params ...bool) *App {
	AsanaApp.Handlers.InsertFilter(pattern, pos, filter, params...)
	return AsanaApp
}
