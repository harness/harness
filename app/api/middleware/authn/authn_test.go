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

package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type fakeAuthenticator struct {
	session *auth.Session
	err     error
}

func (f fakeAuthenticator) Authenticate(*http.Request) (*auth.Session, error) {
	return f.session, f.err
}

func TestAttempt(t *testing.T) {
	user := &auth.Session{Principal: types.Principal{UID: "jane", Type: enum.PrincipalTypeUser}}

	tests := []struct {
		name        string
		authn       fakeAuthenticator
		status      int
		reached     bool
		wantSession *auth.Session
	}{
		{"valid credentials", fakeAuthenticator{session: user}, http.StatusOK, true, user},
		{
			"no credentials falls back to anonymous",
			fakeAuthenticator{err: authn.ErrNoAuthData},
			http.StatusOK, true, &auth.Session{Principal: auth.AnonymousPrincipal},
		},
		{
			"wrapped no-credentials is anonymous",
			fakeAuthenticator{err: errors.Join(errors.New("ctx"), authn.ErrNoAuthData)},
			http.StatusOK, true, &auth.Session{Principal: auth.AnonymousPrincipal},
		},
		{"invalid credentials", fakeAuthenticator{err: errors.New("bad token")}, http.StatusUnauthorized, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				reached bool
				session *auth.Session
			)
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				reached = true
				session, _ = request.AuthSessionFrom(r.Context())
			})

			rec := httptest.NewRecorder()
			Attempt(tt.authn)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, tt.reached, reached)
			require.Equal(t, tt.wantSession, session)
		})
	}
}
