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

package bearer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/remote/clients/registry/auth/null"

	"github.com/stretchr/testify/require"
)

func TestRrmScopeString(t *testing.T) {
	s := &scope{Type: "repository", Name: "lib/img", Actions: []string{"pull", "push"}}
	require.Equal(t, "repository:lib/img:pull,push", s.String())
}

func TestRrmParseScopes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		query  string
		want   int
	}{
		{
			name:   "manifest get",
			method: http.MethodGet,
			path:   "/v2/library/nginx/manifests/latest",
			want:   1,
		},
		{
			name:   "manifest delete",
			method: http.MethodDelete,
			path:   "/v2/library/nginx/manifests/latest",
			want:   1,
		},
		{
			name:   "catalog",
			method: http.MethodGet,
			path:   "/v2/_catalog",
			want:   1,
		},
		{
			name:   "blob upload with mount",
			method: http.MethodPost,
			path:   "/v2/dst/blobs/uploads/",
			query:  "from=src/repo",
			want:   2,
		},
		{
			name:   "unmatched path",
			method: http.MethodGet,
			path:   "/v2/",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(
				context.Background(), tt.method, "http://registry"+tt.path, nil,
			)
			require.NoError(t, err)
			if tt.query != "" {
				req.URL.RawQuery = tt.query
			}
			scopes := parseScopes(req)
			require.Len(t, scopes, tt.want)
		})
	}
}

func TestRrmBearerCache(t *testing.T) {
	c := newCache(2, 0)
	scopes := []*scope{{Type: "repository", Name: "a/b", Actions: []string{"pull"}}}
	tok := &token{Token: "tok1", ExpiresIn: 3600, IssuedAt: time.Now().Format(time.RFC3339)}

	require.Nil(t, c.get(scopes))
	c.set(scopes, tok)
	got := c.get(scopes)
	require.NotNil(t, got)
	require.Equal(t, "tok1", got.Token)
}

func TestRrmBearerCacheExpired(t *testing.T) {
	c := newCache(2, 0)
	scopes := []*scope{{Type: "repository", Name: "a/b", Actions: []string{"pull"}}}
	past := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	tok := &token{Token: "old", ExpiresIn: 60, IssuedAt: past}
	c.set(scopes, tok)
	require.Nil(t, c.get(scopes))
}

func TestRrmGetTokenAccessTokenAlias(t *testing.T) {
	body := []byte(`{"access_token":"from-access","expires_in":60,"issued_at":"2020-01-01T00:00:00Z"}`)
	tok, err := getToken(body, &token{})
	require.NoError(t, err)
	require.Equal(t, "from-access", tok.Token)
}

func TestRrmNewAuthorizerModify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "bearer-tok",
				"expires_in": 3600,
				"issued_at":  time.Now().Format(time.RFC3339),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	auth := NewAuthorizer(srv.URL+"/token", "test-service", null.NewAuthorizer(), srv.Client().Transport)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		srv.URL+"/v2/library/img/manifests/latest",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(req))
	require.Equal(t, fmt.Sprintf("Bearer %s", "bearer-tok"), req.Header.Get("Authorization"))
}

func TestRrmFetchTokenUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`denied`))
	}))
	t.Cleanup(srv.Close)

	a := &authorizer{
		realm:   srv.URL + "/token",
		service: "svc",
		client:  srv.Client(),
	}
	_, err := a.fetchToken(context.Background(), []*scope{{Type: "repository", Name: "a", Actions: []string{"pull"}}})
	require.Error(t, err)
}
