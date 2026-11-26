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

package template

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	templatectrl "github.com/harness/gitness/app/api/controller/template"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/store/database"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func hndTmplServe(t *testing.T, b *hndBase, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext())
	ctx = request.WithAuthSession(ctx, b.session())
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHndTemplateHandlers_badBody(t *testing.T) {
	b := hndNewBase(t)
	ctrl := templatectrl.NewController(hndAuthorizer{mode: hndAllow}, database.NewTemplateStore(b.db), b.spaceFinder)
	for _, h := range []http.HandlerFunc{HandleCreate(ctrl), HandleUpdate(ctrl)} {
		w := hndTmplServe(t, b, h, `{`)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}
