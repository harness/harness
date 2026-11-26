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

package authztest

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestInfAllowAuthorizer(t *testing.T) {
	ctx := context.Background()
	a := AllowAuthorizer{}
	session := &auth.Session{}

	ok, err := a.Check(ctx, session, &types.Scope{}, &types.Resource{}, enum.PermissionRepoView)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = a.CheckAll(ctx, session, types.PermissionCheck{}, types.PermissionCheck{})
	require.NoError(t, err)
	require.True(t, ok)

	results, err := a.CheckMany(ctx, session, types.PermissionCheck{}, types.PermissionCheck{}, types.PermissionCheck{})
	require.NoError(t, err)
	require.Equal(t, []bool{true, true, true}, results)

	results, err = a.CheckMany(ctx, session)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestInfDenyAuthorizer(t *testing.T) {
	ctx := context.Background()
	d := DenyAuthorizer{}
	session := &auth.Session{}

	ok, err := d.Check(ctx, session, &types.Scope{}, &types.Resource{}, enum.PermissionRepoView)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = d.CheckAll(ctx, session, types.PermissionCheck{})
	require.NoError(t, err)
	require.False(t, ok)

	results, err := d.CheckMany(ctx, session, types.PermissionCheck{}, types.PermissionCheck{})
	require.NoError(t, err)
	require.Equal(t, []bool{false, false}, results)
}
