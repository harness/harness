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

package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const rrmAuthPingPath = "/v2/"

func TestRrmNewAuthorizerNonOCI(t *testing.T) {
	auth := NewAuthorizer("user", "pass", true, false)
	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, "http://registry/v2/lib/manifests/latest", nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(req))

	want := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	require.Equal(t, fmt.Sprintf("Basic %s", want), req.Header.Get("Authorization"))
}

func TestRrmNewAuthorizerOCINoChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == rrmAuthPingPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	auth := NewAuthorizer("user", "pass", true, true)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		srv.URL+"/v2/library/img/manifests/latest",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(req))
	require.Empty(t, req.Header.Get("Authorization"))
}

func TestRrmNewAuthorizerOCIBasicChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == rrmAuthPingPath {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	auth := NewAuthorizer("user", "pass", true, true)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		srv.URL+"/v2/library/img/manifests/latest",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(req))
	require.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Basic "))
}

func TestRrmAuthorizerSkipsNonTargetRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == rrmAuthPingPath {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)

	auth := NewAuthorizer("user", "pass", true, true)
	// Initialize with an OCI request first.
	initReq, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, srv.URL+"/v2/x/manifests/latest", nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(initReq))

	// Non-v2 path should not get credentials in OCI mode.
	extReq, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, "https://storage.example/blobs/abc", nil,
	)
	require.NoError(t, err)
	require.NoError(t, auth.Modify(extReq))
	require.Empty(t, extReq.Header.Get("Authorization"))
}
