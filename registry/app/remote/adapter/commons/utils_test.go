//  Copyright 2023 Harness, Inc.
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

package commons

import (
	"context"
	"errors"
	"testing"

	coretypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSpaceResolver is a local fake for spaceResolver.
type stubSpaceResolver struct {
	space    *coretypes.SpaceCore
	err      error
	gotRef   string
	numCalls int
}

func (s *stubSpaceResolver) FindByRef(_ context.Context, ref string) (*coretypes.SpaceCore, error) {
	s.gotRef = ref
	s.numCalls++
	return s.space, s.err
}

func TestSameAccount(t *testing.T) {
	tests := []struct {
		name              string
		registrySpacePath string
		secretSpacePath   string
		wantSame          bool
		wantErr           bool
	}{
		{
			name:              "same_account_same_project",
			registrySpacePath: "acctA/default/proj1",
			secretSpacePath:   "acctA/default/proj1",
			wantSame:          true,
		},
		{
			name:              "same_account_cross_project",
			registrySpacePath: "acctA/default/attacker",
			secretSpacePath:   "acctA/default/victim",
			wantSame:          true,
		},
		{
			name:              "same_account_root_only",
			registrySpacePath: "acctA",
			secretSpacePath:   "acctA/org/proj",
			wantSame:          true,
		},
		{
			name:              "cross_account_rejected",
			registrySpacePath: "attacker-acct/default/proj",
			secretSpacePath:   "victim-acct/default/proj",
			wantSame:          false,
		},
		{
			name:              "cross_account_root_only",
			registrySpacePath: "attacker-acct",
			secretSpacePath:   "victim-acct",
			wantSame:          false,
		},
		{
			name:              "empty_registry_path",
			registrySpacePath: "",
			secretSpacePath:   "acctA/default/proj",
			wantErr:           true,
		},
		{
			name:              "empty_secret_path",
			registrySpacePath: "acctA/default/proj",
			secretSpacePath:   "",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SameAccount(tt.registrySpacePath, tt.secretSpacePath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSame, got)
		})
	}
}

func TestAssertSecretSameAccountAsRegistry(t *testing.T) {
	ctx := context.Background()

	t.Run("nil_secret_space", func(t *testing.T) {
		err := AssertSecretSameAccountAsRegistry(ctx, &stubSpaceResolver{}, "acctA", nil)
		require.Error(t, err)
	})

	t.Run("empty_registry_parent", func(t *testing.T) {
		err := AssertSecretSameAccountAsRegistry(ctx, &stubSpaceResolver{}, "", &coretypes.SpaceCore{Path: "acctA"})
		require.Error(t, err)
	})

	t.Run("finder_error", func(t *testing.T) {
		finder := &stubSpaceResolver{err: errors.New("boom")}
		err := AssertSecretSameAccountAsRegistry(ctx, finder, "acctA", &coretypes.SpaceCore{Path: "acctA/proj"})
		require.Error(t, err)
		assert.Equal(t, "acctA", finder.gotRef)
	})

	t.Run("same_account_ok", func(t *testing.T) {
		finder := &stubSpaceResolver{space: &coretypes.SpaceCore{Path: "acctA/default/attacker"}}
		err := AssertSecretSameAccountAsRegistry(
			ctx, finder, "acctA/default/attacker", &coretypes.SpaceCore{Path: "acctA/default/victim"},
		)
		require.NoError(t, err)
	})

	t.Run("cross_account_rejected", func(t *testing.T) {
		finder := &stubSpaceResolver{space: &coretypes.SpaceCore{Path: "attacker-acct/proj"}}
		err := AssertSecretSameAccountAsRegistry(
			ctx, finder, "attacker-acct/proj", &coretypes.SpaceCore{Path: "victim-acct/proj"},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outside registry account")
	})
}
