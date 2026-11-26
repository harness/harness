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

package serviceaccount

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	principalctrl "github.com/harness/gitness/app/api/controller/principal"
	"github.com/harness/gitness/app/api/request"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHndPrincipalHandlers_missingID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	w := httptest.NewRecorder()
	HandleFind(principalctrl.Controller{}).ServeHTTP(w, r)
	require.NotEqual(t, http.StatusOK, w.Code)

	r = httptest.NewRequest(http.MethodGet, "/", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(request.PathParamPrincipalID, "bad")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
	w = httptest.NewRecorder()
	HandleFind(principalctrl.Controller{}).ServeHTTP(w, r)
	require.NotEqual(t, http.StatusOK, w.Code)
}
