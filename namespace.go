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
	"net/http"
	"strings"

	asanaContext "github.com/goasana/asana/context"
)

type namespaceCond func(*asanaContext.Context) bool

// LinkNamespace used as link action
type LinkNamespace func(*Namespace)

// Namespace is store all the info
type Namespace struct {
	prefix   string
	handlers *ControllerRegister
}

// NewNamespace get new Namespace
func NewNamespace(prefix string, params ...LinkNamespace) *Namespace {
	ns := &Namespace{
		prefix:   prefix,
		handlers: NewControllerRegister(),
	}
	for _, p := range params {
		p(ns)
	}
	return ns
}

// Cond set condition function
// if cond return true can run this namespace, else can't
// usage:
// ns.Cond(func (ctx *context.Context) bool{
//       if ctx.Request.Domain() == "api.asana.me" {
//         return true
//       }
//       return false
//   })
// Cond as the first filter
func (n *Namespace) Cond(cond namespaceCond) *Namespace {
	fn := func(ctx *asanaContext.Context) {
		if !cond(ctx) {
			exception("405", ctx)
		}
	}
	if v := n.handlers.filters[BeforeRouter]; len(v) > 0 {
		mr := new(FilterRouter)
		mr.tree = NewTree()
		mr.pattern = "*"
		mr.filterFunc = fn
		mr.tree.AddRouter("*", true)
		n.handlers.filters[BeforeRouter] = append([]*FilterRouter{mr}, v...)
	} else {
		_ = n.handlers.InsertFilter("*", BeforeRouter, fn)
	}
	return n
}

// Filter add filter in the Namespace
// action has before & after
// FilterFunc
// usage:
// Filter("before", func (ctx *context.Context){
//       _, ok := ctx.Request.Session("uid").(int)
//       if !ok && ctx.HTTPRequest.RequestURI != "/login" {
//          ctx.Redirect(302, "/login")
//        }
//   })
func (n *Namespace) Filter(action string, filter ...FilterFunc) *Namespace {
	var a int
	if action == "before" {
		a = BeforeRouter
	} else if action == "after" {
		a = FinishRouter
	}
	for _, f := range filter {
		n.handlers.InsertFilter("*", a, f)
	}
	return n
}

// Router same as asana.Rourer
// refer: https://godoc.org/github.com/goasana/asana#Router
func (n *Namespace) Router(rootPath string, c ControllerInterface, mappingMethods ...string) *Namespace {
	n.handlers.Add(rootPath, c, mappingMethods...)
	return n
}

// AutoRouter same as asana.AutoRouter
// refer: https://godoc.org/github.com/goasana/asana#AutoRouter
func (n *Namespace) AutoRouter(c ControllerInterface) *Namespace {
	n.handlers.AddAuto(c)
	return n
}

// AutoPrefix same as asana.AutoPrefix
// refer: https://godoc.org/github.com/goasana/asana#AutoPrefix
func (n *Namespace) AutoPrefix(prefix string, c ControllerInterface) *Namespace {
	n.handlers.AddAutoPrefix(prefix, c)
	return n
}

// Get same as asana.Get
// refer: https://godoc.org/github.com/goasana/asana#Get
func (n *Namespace) Get(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Get(rootPath, f)
	return n
}

// Post same as asana.Post
// refer: https://godoc.org/github.com/goasana/asana#Post
func (n *Namespace) Post(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Post(rootPath, f)
	return n
}

// Delete same as asana.Delete
// refer: https://godoc.org/github.com/goasana/asana#Delete
func (n *Namespace) Delete(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Delete(rootPath, f)
	return n
}

// Put same as asana.Put
// refer: https://godoc.org/github.com/goasana/asana#Put
func (n *Namespace) Put(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Put(rootPath, f)
	return n
}

// Head same as asana.Head
// refer: https://godoc.org/github.com/goasana/asana#Head
func (n *Namespace) Head(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Head(rootPath, f)
	return n
}

// Options same as asana.Options
// refer: https://godoc.org/github.com/goasana/asana#Options
func (n *Namespace) Options(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Options(rootPath, f)
	return n
}

// Patch same as asana.Patch
// refer: https://godoc.org/github.com/goasana/asana#Patch
func (n *Namespace) Patch(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Patch(rootPath, f)
	return n
}

// Any same as asana.Any
// refer: https://godoc.org/github.com/goasana/asana#Any
func (n *Namespace) Any(rootPath string, f FilterFunc) *Namespace {
	n.handlers.Any(rootPath, f)
	return n
}

// Handler same as asana.Handler
// refer: https://godoc.org/github.com/goasana/asana#Handler
func (n *Namespace) Handler(rootPath string, h http.Handler) *Namespace {
	n.handlers.Handler(rootPath, h)
	return n
}

// Include add include class
// refer: https://godoc.org/github.com/goasana/asana#Include
func (n *Namespace) Include(cList ...ControllerInterface) *Namespace {
	n.handlers.Include(cList...)
	return n
}

// Namespace add nest Namespace
// usage:
//ns := asana.NewNamespace(“/v1”).
//Namespace(
//    asana.NewNamespace("/shop").
//        Get("/:id", func(ctx *context.Context) {
//            ctx.Body([]byte("shopinfo"))
//    }),
//    asana.NewNamespace("/order").
//        Get("/:id", func(ctx *context.Context) {
//            ctx.Body([]byte("orderinfo"))
//    }),
//    asana.NewNamespace("/crm").
//        Get("/:id", func(ctx *context.Context) {
//            ctx.Body([]byte("crminfo"))
//    }),
//)
func (n *Namespace) Namespace(ns ...*Namespace) *Namespace {
	for _, ni := range ns {
		for k, v := range ni.handlers.routers {
			if _, ok := n.handlers.routers[k]; ok {
				addPrefix(v, ni.prefix)
				n.handlers.routers[k].AddTree(ni.prefix, v)
			} else {
				t := NewTree()
				t.AddTree(ni.prefix, v)
				addPrefix(t, ni.prefix)
				n.handlers.routers[k] = t
			}
		}
		if ni.handlers.enableFilter {
			for pos, filterList := range ni.handlers.filters {
				for _, mr := range filterList {
					t := NewTree()
					t.AddTree(ni.prefix, mr.tree)
					mr.tree = t
					_ = n.handlers.insertFilterRouter(pos, mr)
				}
			}
		}
	}
	return n
}

// AddNamespace register Namespace into asana.Handler
// support multi Namespace
func AddNamespace(nl ...*Namespace) {
	for _, n := range nl {
		for k, v := range n.handlers.routers {
			if _, ok := AsanaApp.Handlers.routers[k]; ok {
				addPrefix(v, n.prefix)
				AsanaApp.Handlers.routers[k].AddTree(n.prefix, v)
			} else {
				t := NewTree()
				t.AddTree(n.prefix, v)
				addPrefix(t, n.prefix)
				AsanaApp.Handlers.routers[k] = t
			}
		}
		if n.handlers.enableFilter {
			for pos, filterList := range n.handlers.filters {
				for _, mr := range filterList {
					t := NewTree()
					t.AddTree(n.prefix, mr.tree)
					mr.tree = t
					_ = AsanaApp.Handlers.insertFilterRouter(pos, mr)
				}
			}
		}
	}
}

func addPrefix(t *Tree, prefix string) {
	for _, v := range t.fixRouters {
		addPrefix(v, prefix)
	}
	if t.wildCard != nil {
		addPrefix(t.wildCard, prefix)
	}
	for _, l := range t.leaves {
		if c, ok := l.runObject.(*ControllerInfo); ok {
			if !strings.HasPrefix(c.pattern, prefix) {
				c.pattern = prefix + c.pattern
			}
		}
	}
}

// NSCond is Namespace Condition
func NSCond(cond namespaceCond) LinkNamespace {
	return func(ns *Namespace) {
		ns.Cond(cond)
	}
}

// NSBefore Namespace BeforeRouter filter
func NSBefore(filterList ...FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Filter("before", filterList...)
	}
}

// NSAfter add Namespace FinishRouter filter
func NSAfter(filterList ...FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Filter("after", filterList...)
	}
}

// NSInclude Namespace Include ControllerInterface
func NSInclude(cList ...ControllerInterface) LinkNamespace {
	return func(ns *Namespace) {
		ns.Include(cList...)
	}
}

// NSRouter call Namespace Router
func NSRouter(rootPath string, c ControllerInterface, mappingMethods ...string) LinkNamespace {
	return func(ns *Namespace) {
		ns.Router(rootPath, c, mappingMethods...)
	}
}

// NSGet call Namespace Get
func NSGet(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Get(rootPath, f)
	}
}

// NSPost call Namespace Post
func NSPost(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Post(rootPath, f)
	}
}

// NSHead call Namespace Head
func NSHead(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Head(rootPath, f)
	}
}

// NSPut call Namespace Put
func NSPut(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Put(rootPath, f)
	}
}

// NSDelete call Namespace Delete
func NSDelete(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Delete(rootPath, f)
	}
}

// NSAny call Namespace Any
func NSAny(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Any(rootPath, f)
	}
}

// NSOptions call Namespace Options
func NSOptions(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Options(rootPath, f)
	}
}

// NSPatch call Namespace Patch
func NSPatch(rootPath string, f FilterFunc) LinkNamespace {
	return func(ns *Namespace) {
		ns.Patch(rootPath, f)
	}
}

// NSAutoRouter call Namespace AutoRouter
func NSAutoRouter(c ControllerInterface) LinkNamespace {
	return func(ns *Namespace) {
		ns.AutoRouter(c)
	}
}

// NSAutoPrefix call Namespace AutoPrefix
func NSAutoPrefix(prefix string, c ControllerInterface) LinkNamespace {
	return func(ns *Namespace) {
		ns.AutoPrefix(prefix, c)
	}
}

// NSNamespace add sub Namespace
func NSNamespace(prefix string, params ...LinkNamespace) LinkNamespace {
	return func(ns *Namespace) {
		n := NewNamespace(prefix, params...)
		ns.Namespace(n)
	}
}

// NSHandler add handler
func NSHandler(rootPath string, h http.Handler) LinkNamespace {
	return func(ns *Namespace) {
		ns.Handler(rootPath, h)
	}
}
