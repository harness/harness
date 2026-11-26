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

package usergroup

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errCxrService = errors.New("service failure")

type cxrService struct {
	usergroup.Service
	space *types.SpaceCore
	err   error
}

func (s *cxrService) List(
	_ context.Context, _ *types.ListQueryFilter, space *types.SpaceCore,
) ([]*types.UserGroupInfo, error) {
	s.space = space
	if s.err != nil {
		return nil, s.err
	}
	return []*types.UserGroupInfo{{Identifier: "devs", Name: "Devs"}}, nil
}

func cxrNewController(b *cxrBase, mode cxrAuthMode, svc usergroup.Service) Controller {
	return *NewController(nil, b.spaces, b.spaceFinder, cxrAuthorizer{mode: mode}, svc)
}

func TestCxrUserGroupList(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t, "root")
	svc := &cxrService{}
	filter := &types.ListQueryFilter{}

	got, err := cxrNewController(b, cxrAllow, svc).List(ctx, b.session(), filter, "root")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, space.ID, svc.space.ID)

	_, err = cxrNewController(b, cxrAllow, svc).List(ctx, b.session(), filter, "nope")
	require.ErrorContains(t, err, "parent space not found")
	_, err = cxrNewController(b, cxrDeny, svc).List(ctx, b.session(), filter, "root")
	require.ErrorContains(t, err, "failed to acquire access to space")
	_, err = cxrNewController(b, cxrAllow, &cxrService{err: errCxrService}).List(ctx, b.session(), filter, "root")
	require.ErrorIs(t, err, errCxrService)
	require.ErrorContains(t, err, "failed to search user groups")
}

func TestCxrUserGroupListScoped(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t, "root")
	svc := &cxrService{}
	filter := &types.ListQueryFilter{}

	got, err := cxrNewController(b, cxrAllow, svc).ListScoped(ctx, b.session(), filter, "root")
	require.NoError(t, err)
	require.Len(t, got, 1)

	_, err = cxrNewController(b, cxrAllow, svc).ListScoped(ctx, b.session(), filter, "")
	require.ErrorContains(t, err, "spacePath is required")
	_, err = cxrNewController(b, cxrAllow, svc).ListScoped(ctx, b.session(), filter, "nope")
	require.ErrorContains(t, err, "failed to acquire access to space")
	_, err = cxrNewController(b, cxrDeny, svc).ListScoped(ctx, b.session(), filter, "root")
	require.ErrorContains(t, err, "failed to acquire access to space")
	_, err = cxrNewController(b, cxrAllow, &cxrService{err: errCxrService}).ListScoped(
		ctx, b.session(), filter, "root")
	require.ErrorIs(t, err, errCxrService)
	require.ErrorContains(t, err, "failed to list user groups")
}
