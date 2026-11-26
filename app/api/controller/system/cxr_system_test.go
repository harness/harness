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

package system

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestCxrSystemIsUserSignupAllowed(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name          string
		existingUsers bool
		enabled       bool
		want          bool
	}{
		{"first user may always sign up", false, false, true},
		{"signup disabled with users", true, false, false},
		{"signup enabled with users", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := cxrNewBase(ctx, t)
			if !tt.existingUsers {
				require.NoError(t, b.principals.DeleteUser(ctx, b.admin.ID))
			}
			c := NewController(b.principals, &types.Config{UserSignupEnabled: tt.enabled})
			got, err := c.IsUserSignupAllowed(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

var errCxrStore = errors.New("store failure")

type cxrFailingPrincipalStore struct {
	store.PrincipalStore
}

func (cxrFailingPrincipalStore) CountUsers(context.Context, *types.UserFilter) (int64, error) {
	return 0, errCxrStore
}

func TestCxrSystemIsUserSignupAllowedStoreError(t *testing.T) {
	c := NewController(cxrFailingPrincipalStore{}, &types.Config{})
	got, err := c.IsUserSignupAllowed(t.Context())
	require.ErrorIs(t, err, errCxrStore)
	require.False(t, got)
}
