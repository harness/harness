// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package router

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/request"
	"github.com/harness/gitness/internal/router/translator"
	"github.com/harness/gitness/internal/store"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

const (
	APIMount           = "/api"
	gitUserAgentPrefix = "git/"
)

type Router struct {
	translator translator.RequestTranslator
	api        http.Handler
	git        http.Handler
	web        http.Handler
}

// NewRouter returns a new http.Handler that routes traffic
// to the appropriate http.Handlers.
func NewRouter(
	translator translator.RequestTranslator,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	tokenStore store.TokenStore,
	saStore store.ServiceAccountStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
) (*Router, error) {
	api := newAPIHandler(systemStore, userStore, spaceStore, repoStore, tokenStore, saStore,
		authenticator, authorizer)
	git := newGitHandler(systemStore, userStore, spaceStore, repoStore, authenticator, authorizer)
	web := newWebHandler(systemStore)

	return &Router{
		translator: translator,
		api:        api,
		git:        git,
		web:        web,
	}, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	// setup logger for request
	log := log.Logger.With().Logger()
	req = req.WithContext(log.WithContext(req.Context()))
	log.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.
			Str("original_url", req.URL.String())
	})

	// Initial translation of the request before any routing.
	req, err = r.translator.TranslatePreRouting(req)
	if err != nil {
		log.Err(err).Msgf("Failed pre-routing translation of request.")
		render.InternalError(w)
		return
	}

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
			return c.Str("handler", "git")
		})

		// Translate git request
		req, err = r.translator.TranslateGit(req)
		if err != nil {
			hlog.FromRequest(req).Err(err).Msgf("Failed GIT translation of request.")
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
	p := req.URL.Path
	if strings.HasPrefix(p, APIMount) {
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("handler", "api")
		})

		// remove matched prefix to simplify API handlers
		if err = stripPrefix(APIMount, req); err != nil {
			hlog.FromRequest(req).Err(err).Msgf("Failed striping of prefix for api request.")
			render.InternalError(w)
			return
		}

		// Translate API request
		req, err = r.translator.TranslateAPI(req)
		if err != nil {
			hlog.FromRequest(req).Err(err).Msgf("Failed API translation of request.")
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
		return c.Str("handler", "web")
	})

	req, err = r.translator.TranslateWeb(req)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msgf("Failed Web translation of request.")
		render.InternalError(w)
		return
	}

	r.web.ServeHTTP(w, req)
}

// stripPrefix removes the prefix from the request path (expected to be there).
func stripPrefix(prefix string, r *http.Request) error {
	return request.ReplacePrefix(r, r.URL.Path[:len(prefix)], "")
}
