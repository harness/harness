// Copyright 2026 Harness, Inc.
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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/controller/principal"
	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/openapi"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/services/usage"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/openapi-go/openapi3"
)

var errCt5AuthFailed = errors.New("authentication failed")

type ct5Authenticator struct {
	session *auth.Session
	err     error
}

func (a ct5Authenticator) Authenticate(*http.Request) (*auth.Session, error) {
	return a.session, a.err
}

type ct5Usage struct{}

func (ct5Usage) Send(context.Context, usage.Metric) error { return nil }

// ct5PrincipalStore implements only the store method the system controller uses.
type ct5PrincipalStore struct {
	store.PrincipalStore
}

func (ct5PrincipalStore) CountUsers(context.Context, *types.UserFilter) (int64, error) { return 1, nil }

type ct5URLs struct {
	url.Provider
	gitHost string
	apiHost string
}

func (u ct5URLs) GetGITHostname(context.Context) string { return u.gitHost }
func (u ct5URLs) GetAPIHostname(context.Context) string { return u.apiHost }

type ct5OpenAPI struct{}

func (ct5OpenAPI) Generate() *openapi3.Spec {
	spec := &openapi3.Spec{Openapi: "3.0.3"}
	spec.Info.WithTitle("ct5").WithVersion("1")
	return spec
}

var _ openapi.Service = ct5OpenAPI{}

func ct5NewAPIHandler(authenticator authn.Authenticator) http.Handler {
	config := &types.Config{}
	config.Cors.AllowedOrigins = []string{"https://ui.example.com"}
	config.Cors.AllowedMethods = []string{http.MethodGet, http.MethodPost}

	return NewAPIHandler(
		context.Background(), config, authenticator,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, principal.Controller{},
		nil, nil, system.NewController(ct5PrincipalStore{}, config), nil, nil, nil, nil, nil, nil,
	)
}

func ct5Do(h http.Handler, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCt5APIHandlerRoutesRequireAuthentication(t *testing.T) {
	handler := ct5NewAPIHandler(ct5Authenticator{err: errCt5AuthFailed})

	routes := []struct{ method, path string }{
		{http.MethodPost, "/v1/logout"},
		{http.MethodGet, "/v1/user"},
		{http.MethodPatch, "/v1/user"},
		{http.MethodGet, "/v1/user/tokens"},
		{http.MethodDelete, "/v1/user/tokens/tok"},
		{http.MethodGet, "/v1/user/sessions"},
		{http.MethodGet, "/v1/user/keys"},
		{http.MethodGet, "/v1/user/favorite"},
		{http.MethodPost, "/v1/spaces"},
		{http.MethodPost, "/v1/spaces/import"},
		{http.MethodGet, "/v1/spaces/root"},
		{http.MethodPatch, "/v1/spaces/root"},
		{http.MethodDelete, "/v1/spaces/root"},
		{http.MethodPost, "/v1/spaces/root/restore"},
		{http.MethodGet, "/v1/spaces/root/spaces"},
		{http.MethodGet, "/v1/spaces/root/repos"},
		{http.MethodGet, "/v1/spaces/root/pipelines"},
		{http.MethodGet, "/v1/spaces/root/members"},
		{http.MethodDelete, "/v1/spaces/root/members/bob"},
		{http.MethodGet, "/v1/spaces/root/labels"},
		{http.MethodGet, "/v1/spaces/root/webhooks"},
		{http.MethodGet, "/v1/spaces/root/rules"},
		{http.MethodGet, "/v1/spaces/root/autolinks"},
		{http.MethodGet, "/v1/spaces/root/usage/metric"},
		{http.MethodGet, "/v1/spaces/root/settings/general"},
		{http.MethodPost, "/v1/repos"},
		{http.MethodPost, "/v1/repos/import"},
		{http.MethodGet, "/v1/repos/root%2Frepo"},
		{http.MethodGet, "/v1/repos/root%2Frepo/summary"},
		{http.MethodGet, "/v1/repos/root%2Frepo/branches"},
		{http.MethodGet, "/v1/repos/root%2Frepo/tags"},
		{http.MethodGet, "/v1/repos/root%2Frepo/commits"},
		{http.MethodGet, "/v1/repos/root%2Frepo/commits/abc123"},
		{http.MethodGet, "/v1/repos/root%2Frepo/content/README.md"},
		{http.MethodGet, "/v1/repos/root%2Frepo/settings/security"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pullreq"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pullreq/1"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pullreq/1/comments"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pullreq/1/reviewers"},
		{http.MethodGet, "/v1/repos/root%2Frepo/webhooks"},
		{http.MethodGet, "/v1/repos/root%2Frepo/rules"},
		{http.MethodGet, "/v1/repos/root%2Frepo/checks/commits/abc123"},
		{http.MethodGet, "/v1/repos/root%2Frepo/labels"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pipelines/pipe"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pipelines/pipe/executions/1"},
		{http.MethodGet, "/v1/repos/root%2Frepo/pipelines/pipe/triggers"},
		{http.MethodGet, "/v1/connectors/conn"},
		{http.MethodGet, "/v1/secrets/sec"},
		{http.MethodGet, "/v1/templates/step/tmpl"},
		{http.MethodGet, "/v1/service-accounts/sa"},
		{http.MethodGet, "/v1/principals"},
		{http.MethodGet, "/v1/principals/1"},
		{http.MethodPost, "/v1/principals/check-emails"},
		{http.MethodGet, "/v1/admin/users/bob"},
		{http.MethodGet, "/v1/gitspaces"},
		{http.MethodGet, "/v1/infraproviders/prov"},
		{http.MethodGet, "/v1/plugins"},
		{http.MethodPost, "/v1/migrate/repos"},
		{http.MethodPost, "/v1/migrate/spaces/root/labels"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := ct5Do(handler, rt.method, rt.path)
			require.Equal(t, http.StatusUnauthorized, rec.Code, "route should exist behind authentication")
		})
	}
}

func TestCt5APIHandlerUnknownRouteIsNotFound(t *testing.T) {
	handler := ct5NewAPIHandler(ct5Authenticator{err: errCt5AuthFailed})

	for _, path := range []string{"/v1/does-not-exist", "/unknown"} {
		require.Equal(t, http.StatusNotFound, ct5Do(handler, http.MethodGet, path).Code, path)
	}
}

func TestCt5APIHandlerPublicRoutes(t *testing.T) {
	handler := ct5NewAPIHandler(ct5Authenticator{err: authn.ErrNoAuthData})

	tests := []struct {
		path string
	}{
		{"/v1/system/health"},
		{"/v1/system/version"},
		{"/v1/resources/gitignore"},
		{"/v1/resources/license"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := ct5Do(handler, http.MethodGet, tt.path)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Header().Get("Cache-Control"), "no-cache")
		})
	}
}

func TestCt5APIHandlerCORS(t *testing.T) {
	handler := ct5NewAPIHandler(ct5Authenticator{err: authn.ErrNoAuthData})

	req := httptest.NewRequest(http.MethodGet, "/v1/system/health", nil)
	req.Header.Set("Origin", "https://ui.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "https://ui.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCt5APIHandlerAnonymousReachesHandler(t *testing.T) {
	// a 200 from the config handler proves the anonymous request passed routing and authentication.
	handler := ct5NewAPIHandler(ct5Authenticator{err: authn.ErrNoAuthData})
	rec := ct5Do(handler, http.MethodGet, "/v1/system/config")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"user_signup_allowed":false`)
}

func TestCt5GitHandler(t *testing.T) {
	provider := ct5URLs{gitHost: "git.example.com", apiHost: "api.example.com"}

	tests := []struct {
		name         string
		nested       bool
		authErr      error
		method       string
		path         string
		wantStatus   int
		wantContains string
	}{
		{
			name: "info refs requires auth route", nested: true, authErr: errCt5AuthFailed,
			method: http.MethodGet, path: "/root/repo.git/info/refs?service=git-upload-pack",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "upload pack route exists", nested: true, authErr: errCt5AuthFailed,
			method: http.MethodPost, path: "/root/repo.git/git-upload-pack", wantStatus: http.StatusUnauthorized,
		},
		{
			name: "lfs batch route exists", nested: false, authErr: errCt5AuthFailed,
			method: http.MethodPost, path: "/root/repo.git/info/lfs/objects/batch", wantStatus: http.StatusUnauthorized,
		},
		{
			name: "dumb protocol is stubbed", nested: true, authErr: authn.ErrNoAuthData,
			method: http.MethodGet, path: "/root/repo.git/HEAD", wantStatus: http.StatusOK,
			wantContains: "asteroid",
		},
		{
			name: "unknown git route", nested: true, authErr: authn.ErrNoAuthData,
			method: http.MethodGet, path: "/root/repo.git/nope", wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &types.Config{NestedSpacesEnabled: tt.nested}
			handler := NewGitHandler(config, provider, ct5Authenticator{err: tt.authErr}, nil, ct5Usage{}, nil)

			rec := ct5Do(handler, tt.method, tt.path)
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.wantContains)
		})
	}
}

func TestCt5StubGitHandler(t *testing.T) {
	rec := ct5Do(stubGitHandler(), http.MethodGet, "/HEAD")
	require.Contains(t, rec.Body.String(), "asteroid")
}

func TestCt5WebHandler(t *testing.T) {
	config := &types.Config{}
	handler := NewWebHandler(
		ct5Authenticator{err: authn.ErrNoAuthData}, ct5OpenAPI{}, NewSecure(config), false, "")

	rec := ct5Do(handler, http.MethodGet, "/openapi.yaml")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/yaml", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "ct5")

	swagger := ct5Do(handler, http.MethodGet, "/swagger/")
	require.Equal(t, http.StatusOK, swagger.Code)
	require.Contains(t, strings.ToLower(swagger.Body.String()), "swagger")
}

func TestCt5NewSecure(t *testing.T) {
	config := &types.Config{}
	config.Secure.FrameDeny = true
	config.Secure.ContentTypeNosniff = true
	config.Secure.ReferrerPolicy = "no-referrer"

	sec := NewSecure(config)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := ct5Do(sec.Handler(next), http.MethodGet, "/")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
}

func TestCt5GetGitRoutingHost(t *testing.T) {
	tests := []struct {
		name string
		git  string
		api  string
		want string
	}{
		{"different hosts", "git.example.com", "api.example.com", "git.example.com"},
		{"same host", "example.com", "example.com", ""},
		{"same host different case", "Example.com", "example.COM", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetGitRoutingHost(context.Background(), ct5URLs{gitHost: tt.git, apiHost: tt.api})
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCt5StripPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	require.NoError(t, StripPrefix("/api", req))
	require.Equal(t, "/v1/user", req.URL.Path)

	other := httptest.NewRequest(http.MethodGet, "/other/path", nil)
	require.NoError(t, StripPrefix("/api", other))
	require.Equal(t, "/other/path", other.URL.Path)
}

type ct5Handler struct {
	name     string
	eligible bool
	handled  *[]string
}

func (h ct5Handler) Handle(http.ResponseWriter, *http.Request) {
	*h.handled = append(*h.handled, h.name)
}
func (h ct5Handler) IsEligibleTraffic(*http.Request) bool { return h.eligible }
func (h ct5Handler) Name() string                         { return h.name }

func TestCt5RouterDispatch(t *testing.T) {
	t.Run("first eligible router handles", func(t *testing.T) {
		var handled []string
		router := NewRouter([]Interface{
			ct5Handler{name: "a", eligible: false, handled: &handled},
			ct5Handler{name: "b", eligible: true, handled: &handled},
			ct5Handler{name: "c", eligible: true, handled: &handled},
		})
		ct5Do(router, http.MethodGet, "/x")
		require.Equal(t, []string{"b"}, handled)
	})

	t.Run("no eligible router", func(t *testing.T) {
		var handled []string
		router := NewRouter([]Interface{ct5Handler{name: "a", handled: &handled}})
		rec := ct5Do(router, http.MethodGet, "/x")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, handled)
	})
}

func TestCt5GitRouter(t *testing.T) {
	var gotPath string
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { gotPath = r.URL.Path })

	r := NewGitRouter(inner, "git.example.com")
	require.Equal(t, "git", r.Name())

	tests := []struct {
		name string
		path string
		host string
		want bool
	}{
		{"git mount prefix", "/git/root/repo.git/info/refs", "api.example.com", true},
		{"git host", "/root/repo.git/info/refs", "git.example.com", true},
		{"git host with port", "/root/repo.git/info/refs", "GIT.example.com:8443", true},
		{"other host", "/root/repo.git/info/refs", "api.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Host = tt.host
			require.Equal(t, tt.want, r.IsEligibleTraffic(req))
		})
	}

	t.Run("no git host configured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/root/repo.git", nil)
		req.Host = "git.example.com"
		require.False(t, NewGitRouter(inner, "").IsEligibleTraffic(req))
	})

	t.Run("handle strips mount", func(t *testing.T) {
		r.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/git/root/repo.git", nil))
		require.Equal(t, "/root/repo.git", gotPath)
	})
}

func TestCt5APIRouter(t *testing.T) {
	var gotPath string
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { gotPath = r.URL.Path })

	r := NewAPIRouter(inner)
	require.Equal(t, "api", r.Name())
	require.True(t, r.IsEligibleTraffic(httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)))
	require.False(t, r.IsEligibleTraffic(httptest.NewRequest(http.MethodGet, "/apix", nil)))

	r.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/user", nil))
	require.Equal(t, "/v1/user", gotPath)
}

func TestCt5WebRouter(t *testing.T) {
	called := false
	r := NewWebRouter(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))

	require.Equal(t, "web", r.Name())
	require.True(t, r.IsEligibleTraffic(httptest.NewRequest(http.MethodGet, "/anything", nil)))

	r.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	require.True(t, called)
}

func TestCt5WithLoggingRouter(t *testing.T) {
	require.NotNil(t, WithLoggingRouter("api"))
}

func TestCt5ProvideRouter(t *testing.T) {
	config := &types.Config{}
	registry := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })

	router := ProvideRouter(
		context.Background(), config, ct5Authenticator{err: errCt5AuthFailed},
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, principal.Controller{},
		nil, nil, nil, nil, nil, nil, nil, nil,
		ct5URLs{gitHost: "git.example.com", apiHost: "api.example.com"}, ct5OpenAPI{}, registry, ct5Usage{}, nil,
	)
	require.Len(t, router.routers, 4)

	tests := []struct {
		name       string
		target     string
		host       string
		wantStatus int
	}{
		{"registry traffic", "/v2/", "api.example.com", http.StatusTeapot},
		{"api traffic", "/api/v1/user", "api.example.com", http.StatusUnauthorized},
		{"git host traffic", "/root/repo.git/info/refs", "git.example.com", http.StatusUnauthorized},
		{"web traffic", "/openapi.yaml", "api.example.com", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
