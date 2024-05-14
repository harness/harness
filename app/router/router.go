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

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	APIMount = "/api"
	GitMount = "/git"
)

type Router struct {
	api APIHandler
	git GitHandler
	web WebHandler

	// gitHost describes the optional host via which git traffic is identified.
	// Note: always stored as lowercase.
	gitHost string
}

// NewRouter returns a new http.Handler that routes traffic
// to the appropriate handlers.
func NewRouter(
	api APIHandler,
	git GitHandler,
	web WebHandler,
	gitHost string,
) *Router {
	return &Router{
		api: api,
		git: git,
		web: web,

		gitHost: strings.ToLower(gitHost),
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
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

	/*
	 * 1. GIT
	 *
	 * All Git originating traffic starts with "/space1/space2/repo.git".
	 */
	if r.isGitTraffic(req) {
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("http.handler", "git")
		})

		// remove matched prefix to simplify API handlers (only if it's there)
		if err = stripPrefix(GitMount, req); err != nil {
			log.Err(err).Msgf("Failed striping of prefix for git request.")
			render.InternalError(ctx, w)
			return
		}

		r.git.ServeHTTP(w, req)
		return
	}

	/*
	 * 2. REST API
	 *
	 * All Rest API calls start with "/api/", and thus can be uniquely identified.
	 */
	if r.isAPITraffic(req) {
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("http.handler", "api")
		})

		// remove matched prefix to simplify API handlers
		if err = stripPrefix(APIMount, req); err != nil {
			log.Err(err).Msgf("Failed striping of prefix for api request.")
			render.InternalError(ctx, w)
			return
		}

		r.api.ServeHTTP(w, req)
		return
	}

	/*
	 * 3. WEB
	 *
	 * Everything else will be routed to web (or return 404)
	 */
	log.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("http.handler", "web")
	})

	r.web.ServeHTTP(w, req)
}

// stripPrefix removes the prefix from the request path (or noop if it's not there).
func stripPrefix(prefix string, req *http.Request) error {
	if !strings.HasPrefix(req.URL.Path, prefix) {
		return nil
	}
	return request.ReplacePrefix(req, prefix, "")
}

// isGitTraffic returns true iff the request is identified as part of the git http protocol.
func (r *Router) isGitTraffic(req *http.Request) bool {
	// git traffic is always reachable via the git mounting path.
	if strings.HasPrefix(req.URL.Path, GitMount+"/") {
		return true
	}

	// otherwise check if the request came in via the configured git host (if enabled)
	if len(r.gitHost) > 0 {
		// cut (optional) port off the host
		h, _, _ := strings.Cut(req.Host, ":")

		// check if request host matches the configured git host (case insensitive)
		if r.gitHost == strings.ToLower(h) {
			return true
		}
	}

	// otherwise we don't treat it as git traffic
	return false
}

// isAPITraffic returns true iff the request is identified as part of our rest API.
func (r *Router) isAPITraffic(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, APIMount+"/")
}
