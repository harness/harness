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

package secret

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	secretctrl "github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/encrypt"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func hndSecretServe(
	t *testing.T, b *hndBase, h http.HandlerFunc, body string, params map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	for k, v := range params {
		routeCtx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)
	ctx = request.WithAuthSession(ctx, b.session())
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHndSecretHandlers_badBody(t *testing.T) {
	b := hndNewBase(t)
	enc, err := encrypt.New(strings.Repeat("k", 32), false)
	require.NoError(t, err)
	ctrl := secretctrl.NewController(hndAuthorizer{mode: hndAllow}, enc, database.NewSecretStore(b.db), b.spaceFinder)

	for _, h := range []http.HandlerFunc{HandleCreate(ctrl), HandleUpdate(ctrl)} {
		w := hndSecretServe(t, b, h, `{`, nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}

	w := hndSecretServe(t, b, HandleFind(ctrl), "", map[string]string{
		request.PathParamSecretRef: "root/missing",
	})
	require.NotEqual(t, http.StatusOK, w.Code)
}
