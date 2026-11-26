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

package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/web"

	"github.com/stretchr/testify/require"
)

type fakeAuthenticator struct {
	err error
}

func (f fakeAuthenticator) Authenticate(*http.Request) (*auth.Session, error) {
	if f.err != nil {
		return nil, f.err
	}

	return &auth.Session{}, nil
}

func TestPublicAccess(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		authErr       error
		wantPublicCtx bool
	}{
		{"disabled", false, authn.ErrNoAuthData, false},
		{"enabled without credentials", true, authn.ErrNoAuthData, true},
		{"enabled with valid credentials", true, nil, false},
		{"enabled with invalid credentials", true, errors.New("bad token"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got, called bool
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				called = true
				got = web.RenderPublicAccessFrom(r.Context())
			})

			h := PublicAccess(tt.enabled, fakeAuthenticator{err: tt.authErr})(next)
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			require.True(t, called)
			require.Equal(t, tt.wantPublicCtx, got)
		})
	}
}
