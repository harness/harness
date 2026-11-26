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

package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHndHLogRequestIDHandler(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		id, ok := request.RequestIDFrom(r.Context())
		require.True(t, ok)
		require.NotEmpty(t, id)
	})

	r := httptest.NewRequest(http.MethodGet, "/api", nil)
	r.Header.Set(requestIDHeader, "req-hnd-1")
	w := httptest.NewRecorder()
	HLogRequestIDHandler()(next).ServeHTTP(w, r)
	require.True(t, called)
	require.Equal(t, "req-hnd-1", w.Header().Get(requestIDHeader))
}

func TestHndHLogRequestIDHandler_generatesID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := request.RequestIDFrom(r.Context())
		require.True(t, ok)
	})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	HLogRequestIDHandler()(next).ServeHTTP(w, r)
	require.NotEmpty(t, w.Header().Get(requestIDHeader))
}

func TestHndURLHandler(t *testing.T) {
	ctx := zerolog.New(nil).With().Logger().WithContext(context.Background())
	r := httptest.NewRequest(http.MethodGet, "/x/y", nil).WithContext(ctx)
	r.URL.RawPath = "/raw"
	next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	URLHandler("url")(next).ServeHTTP(w, r)
	require.Equal(t, http.StatusNoContent, w.Code)
}
