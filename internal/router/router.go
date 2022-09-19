// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package router

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
)

const (
	restMount          = "/api"
	gitUserAgentPrefix = "git/"
)

type Router struct {
	api http.Handler
	git http.Handler
	web http.Handler
}

// New returns a new http.Handler that routes traffic
// to the appropriate http.Handlers.
func New(
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
) (http.Handler, error) {
	api := newAPIHandler(restMount, systemStore, userStore, spaceStore, repoStore, authenticator, authorizer)
	git := newGitHandler("/", systemStore, userStore, spaceStore, repoStore, authenticator, authorizer)
	web := newWebHandler("/", systemStore)

	return &Router{
		api: api,
		git: git,
		web: web,
	}, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	/*
	 * 1. GIT
	 *
	 * All Git originating traffic starts with "/space1/space2/repo.git".
	 * This can collide with other API endpoints and thus has to be checked first.
	 * To avoid any false positives, we use the user-agent header to identify git agents.
	 */
	ua := req.Header.Get("user-agent")
	if strings.HasPrefix(ua, gitUserAgentPrefix) {
		r.git.ServeHTTP(w, req)
		return
	}

	/*
	 * 2. REST API
	 *
	 * All Rest API calls start with "/api/", and thus can be uniquely identified.
	 * Note: This assumes that we are blocking "api" as a space name!
	 */
	p := req.URL.Path
	if strings.HasPrefix(p, restMount) {
		r.api.ServeHTTP(w, req)
		return
	}

	/*
	 * 3. WEB
	 *
	 * Everything else will be routed to web (or return 404)
	 */
	r.web.ServeHTTP(w, req)
}
