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

package usergroup

import (
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	svc := NewService()

	groups, err := svc.List(t.Context(), &types.ListQueryFilter{}, &types.SpaceCore{})
	require.Error(t, err)
	require.Nil(t, groups)

	ids, err := svc.ListUserIDsByGroupIDs(t.Context(), []int64{1, 2})
	require.NoError(t, err)
	require.Empty(t, ids)
	require.NotNil(t, ids)

	principals, err := svc.MapGroupIDsToPrincipals(t.Context(), []int64{1})
	require.NoError(t, err)
	require.Empty(t, principals)
	require.NotNil(t, principals)
}

func TestGitnessResolver(t *testing.T) {
	group, err := NewGitnessResolver().Resolve(t.Context(), "space/group")
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, group)
}
