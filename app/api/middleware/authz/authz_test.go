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

package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestBlockSessionToken(t *testing.T) {
	withMetadata := func(md auth.Metadata) context.Context {
		return request.WithAuthSession(context.Background(), &auth.Session{Metadata: md})
	}

	tests := []struct {
		name    string
		ctx     context.Context
		blocked bool
	}{
		{"no session", context.Background(), false},
		{"session without metadata", withMetadata(nil), false},
		{"session token", withMetadata(&auth.TokenMetadata{TokenType: enum.TokenTypeSession}), true},
		{"personal access token", withMetadata(&auth.TokenMetadata{TokenType: enum.TokenTypePAT}), false},
		{"service account token", withMetadata(&auth.TokenMetadata{TokenType: enum.TokenTypeSAT}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })

			rec := httptest.NewRecorder()
			BlockSessionToken(next).ServeHTTP(rec,
				httptest.NewRequest(http.MethodGet, "/", nil).WithContext(tt.ctx))

			require.Equal(t, !tt.blocked, reached)
			if tt.blocked {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			} else {
				require.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}
