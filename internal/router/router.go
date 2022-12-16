// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package router

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/request"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

const (
	APIMount           = "/api"
	gitUserAgentPrefix = "git/"
)

type Router struct {
	api APIHandler
	git GitHandler
	web WebHandler
}

// NewRouter returns a new http.Handler that routes traffic
// to the appropriate handlers.
func NewRouter(
	api APIHandler,
	git GitHandler,
	web WebHandler,
) *Router {
	return &Router{
		api: api,
		git: git,
		web: web,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	// setup logger for request
	log := log.Logger.With().Logger()
	req = req.WithContext(log.WithContext(req.Context()))
	log.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.
			Str("http.original_url", req.URL.String())
	})

	/*
	 * 1. GIT
	 *
	 * All Git originating traffic starts with "/space1/space2/repo.git".
	 * This can collide with other API endpoints and thus has to be checked first.
	 * To avoid any false positives, we use the user-agent header to identify git agents.
	 */
	ua := req.Header.Get("user-agent")
	if strings.HasPrefix(ua, gitUserAgentPrefix) {
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("http.handler", "git")
		})

		r.git.ServeHTTP(w, req)
		return
	}

	/*
	 * 2. REST API
	 *
	 * All Rest API calls start with "/api/", and thus can be uniquely identified.
	 */
	p := req.URL.Path
	if strings.HasPrefix(p, APIMount) {
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("http.handler", "api")
		})

		// remove matched prefix to simplify API handlers
		if err = stripPrefix(APIMount, req); err != nil {
			hlog.FromRequest(req).Err(err).Msgf("Failed striping of prefix for api request.")
			render.InternalError(w)
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

// stripPrefix removes the prefix from the request path (expected to be there).
func stripPrefix(prefix string, r *http.Request) error {
	return request.ReplacePrefix(r, r.URL.Path[:len(prefix)], "")
}
