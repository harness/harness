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

package native

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/remote/clients/registry/auth/null"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func rrmNewNativeAdapter(t *testing.T, handler http.Handler) *Adapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewAdapterWithAuthorizer(types.UpstreamProxy{
		RepoURL:     srv.URL,
		PackageType: artifact.PackageTypeDOCKER,
	}, null.NewAuthorizer())
}

func TestRrmNativeAdapterPingSimple(t *testing.T) {
	a := rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, a.PingSimple(context.Background()))

	a = rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	require.NoError(t, a.PingSimple(context.Background()))

	a = rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	require.NoError(t, a.PingSimple(context.Background()))

	a = rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	require.Error(t, a.PingSimple(context.Background()))
}

func TestRrmNativeAdapterStubs(t *testing.T) {
	a := rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	msg, err := a.HealthCheck()
	require.NoError(t, err)
	require.Equal(t, "Not implemented", msg)

	require.Error(t, a.DeleteTag(context.Background(), "repo", "tag"))

	mount, repo, err := a.CanBeMount(context.Background(), "sha256:abc")
	require.NoError(t, err)
	require.False(t, mount)
	require.Empty(t, repo)

	name, err := a.GetImageName("library/nginx")
	require.NoError(t, err)
	require.Equal(t, "library/nginx", name)
}

func TestRrmNativeAdapterPingSimpleUnauthorizedErrorType(t *testing.T) {
	a := rrmNewNativeAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	err := a.Ping(context.Background())
	require.True(t, errors.IsErr(err, errors.UnAuthorizedCode))
}
