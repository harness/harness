// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/request"
	"github.com/harness/gitness/logging"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Router struct {
	routers []Interface
}

// NewRouter returns a new http.Handler that routes traffic
// to the appropriate handlers.
func NewRouter(
	routers []Interface,
) *Router {
	return &Router{
		routers: routers,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// setup logger for request
	log := log.Logger.With().Logger()
	ctx := log.WithContext(req.Context())
	// add logger to logr interface for usage in 3rd party libs
	ctx = logr.NewContext(ctx, zerologr.New(&log))
	req = req.WithContext(ctx)
	log.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.
			Str("http.original_url", req.URL.String())
	})

	for _, router := range r.routers {
		if ok := router.IsEligibleTraffic(req); ok {
			req = req.WithContext(logging.NewContext(req.Context(), WithLoggingRouter(router.Name())))
			router.Handle(w, req)
			return
		}
	}
	render.BadRequestf(ctx, w, "No eligible router found")
}

// StripPrefix removes the prefix from the request path (or noop if it's not there).
func StripPrefix(prefix string, req *http.Request) error {
	if !strings.HasPrefix(req.URL.Path, prefix) {
		return nil
	}
	return request.ReplacePrefix(req, prefix, "")
}
