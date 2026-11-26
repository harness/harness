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

package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/usererror"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// newTestRequest creates a GET request for the url (path and query) with the provided chi path parameters.
func newTestRequest(t *testing.T, target string, pathParams ...string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	return withPathParams(req, pathParams...)
}

// withPathParams stores the key/value pairs as chi URL parameters of the request.
func withPathParams(req *http.Request, pathParams ...string) *http.Request {
	rctx := chi.NewRouteContext()
	for i := 0; i+1 < len(pathParams); i += 2 {
		rctx.URLParams.Add(pathParams[i], pathParams[i+1])
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// requireBadRequest asserts that err is a user facing 400 error containing the message fragment.
func requireBadRequest(t *testing.T, err error, fragment string) {
	t.Helper()

	require.Error(t, err)
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, http.StatusBadRequest, userErr.Status)
	require.Contains(t, userErr.Message, fragment)
}
