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

package goget

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHndGogetMiddleware_passThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	mw := Middleware(3, nil, nil)(next)
	r := httptest.NewRequest(http.MethodGet, "/import/path", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	require.Equal(t, http.StatusTeapot, w.Code)
}

func TestHndGogetMiddleware_missingRepoRef(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("next should not run") })
	r := httptest.NewRequest(http.MethodGet, "/?go-get=1", nil)
	w := httptest.NewRecorder()
	Middleware(3, nil, nil)(next).ServeHTTP(w, r)
	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestHndGogetMiddleware_postIgnored(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	r := httptest.NewRequest(http.MethodPost, "/?go-get=1", nil)
	w := httptest.NewRecorder()
	Middleware(3, nil, nil)(next).ServeHTTP(w, r)
	require.True(t, called)
}
