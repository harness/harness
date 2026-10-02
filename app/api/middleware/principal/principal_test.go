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

package principal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestGitHookRestrict(t *testing.T) {
	tests := []struct {
		name             string
		principal        *types.Principal
		wantNext         bool
		wantResponseBody string
	}{
		{
			name:             "missing principal",
			wantResponseBody: `{"error":"Internal error"}`,
		},
		{
			name: "anonymous principal",
			principal: &types.Principal{
				ID:   auth.AnonymousPrincipal.ID,
				UID:  auth.AnonymousPrincipal.UID,
				Type: enum.PrincipalTypeUser,
			},
			wantResponseBody: `{"error":"Internal error"}`,
		},
		{
			name: "user principal",
			principal: &types.Principal{
				ID:   1,
				UID:  "user",
				Type: enum.PrincipalTypeUser,
			},
			wantResponseBody: `{"error":"Internal error"}`,
		},
		{
			name: "service principal",
			principal: &types.Principal{
				ID:   1,
				UID:  "service",
				Type: enum.PrincipalTypeService,
			},
			wantNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusNoContent)
			})
			handler := GitHookRestrict()(next)

			req := httptest.NewRequest(http.MethodPost, "/v1/internal/git-hooks/pre-receive", nil)
			if tt.principal != nil {
				req = req.WithContext(request.WithAuthSession(req.Context(), &auth.Session{
					Principal: *tt.principal,
				}))
			}
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tt.wantNext, nextCalled)
			if tt.wantResponseBody != "" {
				require.JSONEq(t, tt.wantResponseBody, resp.Body.String())
				require.Equal(t, http.StatusOK, resp.Code)
			} else {
				require.Equal(t, http.StatusNoContent, resp.Code)
			}
		})
	}
}
