// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-swagger/go-swagger/errors"
	"github.com/go-swagger/go-swagger/httpkit"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/strfmt"
	"github.com/gorilla/context"
	"github.com/naoina/denco"
)

// RouteParam is a object to capture route params in a framework agnostic way.
// implementations of the muxer should use these route params to communicate with the
// swagger framework
type RouteParam struct {
	Name  string
	Value string
}

// RouteParams the collection of route params
type RouteParams []RouteParam

// Get gets the value for the route param for the specified key
func (r RouteParams) Get(name string) string {
	vv, _, _ := r.GetOK(name)
	if len(vv) > 0 {
		return vv[len(vv)-1]
	}
	return ""
}

// GetOK gets the value but also returns booleans to indicate if a key or value
// is present. This aids in validation and satisfies an interface in use there
//
// The returned values are: data, has key, has value
func (r RouteParams) GetOK(name string) ([]string, bool, bool) {
	for _, p := range r {
		if p.Name == name {
			return []string{p.Value}, true, p.Value != ""
		}
	}
	return nil, false, false
}

func newRouter(ctx *Context, next http.Handler) http.Handler {
	if ctx.router == nil {
		ctx.router = DefaultRouter(ctx.spec, ctx.api)
	}
	basePath := ctx.spec.BasePath()
	isRoot := basePath == "" || basePath == "/"
	for strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		defer context.Clear(r)
		// use context to lookup routes
		if isRoot {
			if _, ok := ctx.RouteInfo(r); ok {
				next.ServeHTTP(rw, r)
				return
			}
		} else {
			if p := strings.TrimPrefix(r.URL.Path, basePath); len(p) < len(r.URL.Path) {
				r.URL.Path = p
				if _, ok := ctx.RouteInfo(r); ok {
					next.ServeHTTP(rw, r)
					return
				}
			}
		}
		// Not found, check if it exists in the other methods first
		if others := ctx.AllowedMethods(r); len(others) > 0 {
			ctx.Respond(rw, r, ctx.spec.RequiredProduces(), nil, errors.MethodNotAllowed(r.Method, others))
			return
		}

		ctx.Respond(rw, r, ctx.spec.RequiredProduces(), nil, errors.NotFound("path %s was not found", r.URL.Path))
	})
}

// RoutableAPI represents an interface for things that can serve
// as a provider of implementations for the swagger router
type RoutableAPI interface {
	HandlerFor(string, string) (http.Handler, bool)
	ServeErrorFor(string) func(http.ResponseWriter, *http.Request, error)
	ConsumersFor([]string) map[string]httpkit.Consumer
	ProducersFor([]string) map[string]httpkit.Producer
	AuthenticatorsFor(map[string]spec.SecurityScheme) map[string]httpkit.Authenticator
	Formats() strfmt.Registry
	DefaultProduces() string
	DefaultConsumes() string
}

// Router represents a swagger aware router
type Router interface {
	Lookup(method, path string) (*MatchedRoute, bool)
	OtherMethods(method, path string) []string
}

type defaultRouteBuilder struct {
	spec    *spec.Document
	api     RoutableAPI
	records map[string][]denco.Record
}

type defaultRouter struct {
	spec    *spec.Document
	api     RoutableAPI
	routers map[string]*denco.Router
}

func newDefaultRouteBuilder(spec *spec.Document, api RoutableAPI) *defaultRouteBuilder {
	return &defaultRouteBuilder{
		spec:    spec,
		api:     api,
		records: make(map[string][]denco.Record),
	}
}

// DefaultRouter creates a default implemenation of the router
func DefaultRouter(spec *spec.Document, api RoutableAPI) Router {
	builder := newDefaultRouteBuilder(spec, api)
	if spec != nil {
		for method, paths := range spec.Operations() {
			for path, operation := range paths {
				builder.AddRoute(method, path, operation)
			}
		}
	}
	return builder.Build()
}

type routeEntry struct {
	PathPattern    string
	BasePath       string
	Operation      *spec.Operation
	Consumes       []string
	Consumers      map[string]httpkit.Consumer
	Produces       []string
	Producers      map[string]httpkit.Producer
	Parameters     map[string]spec.Parameter
	Handler        http.Handler
	Formats        strfmt.Registry
	Binder         *untypedRequestBinder
	Authenticators map[string]httpkit.Authenticator
}

// MatchedRoute represents the route that was matched in this request
type MatchedRoute struct {
	routeEntry
	Params   RouteParams
	Consumer httpkit.Consumer
	Producer httpkit.Producer
}

func (d *defaultRouter) Lookup(method, path string) (*MatchedRoute, bool) {
	if router, ok := d.routers[strings.ToUpper(method)]; ok {
		if m, rp, ok := router.Lookup(path); ok && m != nil {
			if entry, ok := m.(*routeEntry); ok {
				var params RouteParams
				for _, p := range rp {
					params = append(params, RouteParam{Name: p.Name, Value: p.Value})
				}
				return &MatchedRoute{routeEntry: *entry, Params: params}, true
			}
		}
	}
	return nil, false
}

func (d *defaultRouter) OtherMethods(method, path string) []string {
	mn := strings.ToUpper(method)
	var methods []string
	for k, v := range d.routers {
		if k != mn {
			if _, _, ok := v.Lookup(path); ok {
				methods = append(methods, k)
				continue
			}
		}
	}
	return methods
}

var pathConverter = regexp.MustCompile(`{(\w+)}`)

func (d *defaultRouteBuilder) AddRoute(method, path string, operation *spec.Operation) {
	mn := strings.ToUpper(method)

	if handler, ok := d.api.HandlerFor(method, path); ok {
		consumes := d.spec.ConsumesFor(operation)
		produces := d.spec.ProducesFor(operation)
		parameters := d.spec.ParamsFor(method, path)
		definitions := d.spec.SecurityDefinitionsFor(operation)

		record := denco.NewRecord(pathConverter.ReplaceAllString(path, ":$1"), &routeEntry{
			Operation:      operation,
			Handler:        handler,
			Consumes:       consumes,
			Produces:       produces,
			Consumers:      d.api.ConsumersFor(consumes),
			Producers:      d.api.ProducersFor(produces),
			Parameters:     parameters,
			Formats:        d.api.Formats(),
			Binder:         newUntypedRequestBinder(parameters, d.spec.Spec(), d.api.Formats()),
			Authenticators: d.api.AuthenticatorsFor(definitions),
		})
		d.records[mn] = append(d.records[mn], record)
	}
}

func (d *defaultRouteBuilder) Build() *defaultRouter {
	routers := make(map[string]*denco.Router)
	for method, records := range d.records {
		router := denco.New()
		router.Build(records)
		routers[method] = router
	}
	return &defaultRouter{
		spec:    d.spec,
		routers: routers,
	}
}
