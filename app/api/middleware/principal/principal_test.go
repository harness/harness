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

package principal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func withPrincipal(ctx context.Context, p types.Principal) context.Context {
	return request.WithAuthSession(ctx, &auth.Session{Principal: p})
}

// run executes the middleware around a handler and reports whether the handler was reached.
func run(
	t *testing.T,
	mw func(http.Handler) http.Handler,
	ctx context.Context,
) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	reached := false
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))

	return rec, reached
}

func TestRestrictTo(t *testing.T) {
	user := types.Principal{UID: "jane", Type: enum.PrincipalTypeUser}
	service := types.Principal{UID: "svc", Type: enum.PrincipalTypeService}

	tests := []struct {
		name    string
		ctx     context.Context
		status  int
		reached bool
	}{
		{"no session", context.Background(), http.StatusForbidden, false},
		{"anonymous", withPrincipal(context.Background(), auth.AnonymousPrincipal), http.StatusUnauthorized, false},
		{"wrong type", withPrincipal(context.Background(), service), http.StatusForbidden, false},
		{"matching type", withPrincipal(context.Background(), user), http.StatusOK, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, reached := run(t, RestrictTo(enum.PrincipalTypeUser), tt.ctx)
			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.reached, reached)
		})
	}
}

func TestRestrictToAdmin(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		status  int
		reached bool
	}{
		{"no session", context.Background(), http.StatusForbidden, false},
		{"non admin", withPrincipal(context.Background(), types.Principal{UID: "jane"}), http.StatusForbidden, false},
		{"admin", withPrincipal(context.Background(), types.Principal{UID: "root", Admin: true}), http.StatusOK, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, reached := run(t, RestrictToAdmin(), tt.ctx)
			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.reached, reached)
		})
	}
}

func TestGitHookRestrict(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		blocked bool
	}{
		{"no session", context.Background(), true},
		{"anonymous", withPrincipal(context.Background(), auth.AnonymousPrincipal), true},
		{"user", withPrincipal(context.Background(), types.Principal{UID: "jane", Type: enum.PrincipalTypeUser}), true},
		{"service", withPrincipal(context.Background(), types.Principal{UID: "svc", Type: enum.PrincipalTypeService}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, reached := run(t, GitHookRestrict(), tt.ctx)

			// Git hooks always get a 200 so the hook client can read the error payload.
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, !tt.blocked, reached)
			if tt.blocked {
				require.Contains(t, rec.Body.String(), "Internal error")
			} else {
				require.Empty(t, rec.Body.String())
			}
		})
	}
}
