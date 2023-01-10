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
	req = req.WithContext(log.WithContext(req.Context()))
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
			hlog.FromRequest(req).Err(err).Msgf("Failed striping of prefix for git request.")
			render.InternalError(w)
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

// stripPrefix removes the prefix from the request path (or noop if it's not there).
func stripPrefix(prefix string, req *http.Request) error {
	p := req.URL.Path
	if !strings.HasPrefix(p, prefix) {
		return nil
	}
	return request.ReplacePrefix(req, req.URL.Path[:len(prefix)], "")
}

// isGitTraffic returns true iff the request is identified as part of the git http protocol.
func (r *Router) isGitTraffic(req *http.Request) bool {
	// git traffic is always reachable via the git mounting path.
	p := req.URL.Path
	if strings.HasPrefix(p, GitMount) {
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
	p := req.URL.Path
	return strings.HasPrefix(p, APIMount)
}
