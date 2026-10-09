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

package pullreq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/controller/check"
	"github.com/harness/gitness/app/api/request"

	"github.com/go-chi/chi/v5"
)

func TestHandleListCheckSummaries_malformedBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{"commit_shas": [`},
		{name: "wrong type", body: `{"commit_shas": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add(request.PathParamRepoRef, "space/repo")

			r := httptest.NewRequest(http.MethodPost, "/repos/space%2Frepo/checks/summary",
				strings.NewReader(tt.body))
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
			w := httptest.NewRecorder()

			HandleListCheckSummaries(&check.Controller{}).ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}

			var resp struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if !strings.HasPrefix(resp.Message, "Invalid request body") {
				t.Fatalf("message = %q, want it to start with %q", resp.Message, "Invalid request body")
			}
		})
	}
}
