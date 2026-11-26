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

package swagger_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/registry/app/api/handler/swagger"

	"github.com/stretchr/testify/require"
)

func TestRchSwaggerHandler(t *testing.T) {
	h := swagger.GetSwaggerHandler("/api/v1")

	t.Run("serves the openapi document", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/swagger.json", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var doc map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
		require.Contains(t, doc, "openapi")
		require.Contains(t, doc, "paths")
	})
	t.Run("serves the swagger ui", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/swagger/index.html", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "swagger")
	})
	t.Run("unknown paths are not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/other", nil))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
