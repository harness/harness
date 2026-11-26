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

package pullreq

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type countSpaceStore struct {
	store.SpaceStore

	descendants []types.SpaceParentData
	err         error
}

func (f countSpaceStore) GetDescendantsData(context.Context, int64) ([]types.SpaceParentData, error) {
	return f.descendants, f.err
}

type countPullReqStore struct {
	store.PullReqStore

	count    int64
	err      error
	gotSpace []int64
}

func (f *countPullReqStore) Count(_ context.Context, filter *types.PullReqFilter) (int64, error) {
	f.gotSpace = filter.SpaceIDs
	return f.count, f.err
}

func TestListService_CountForSpace(t *testing.T) {
	space := &types.SpaceCore{ID: 10}

	t.Run("only the space itself", func(t *testing.T) {
		prs := &countPullReqStore{count: 4}
		svc := &ListService{pullreqStore: prs, spaceStore: countSpaceStore{err: errors.New("must not be called")}}

		filter := &types.PullReqFilter{}
		count, err := svc.CountForSpace(t.Context(), space, false, filter)
		require.NoError(t, err)
		require.Equal(t, int64(4), count)
		require.Equal(t, []int64{10}, prs.gotSpace)
		require.Equal(t, []int64{10}, filter.SpaceIDs)
	})

	t.Run("including subspaces", func(t *testing.T) {
		prs := &countPullReqStore{count: 9}
		svc := &ListService{
			pullreqStore: prs,
			spaceStore: countSpaceStore{descendants: []types.SpaceParentData{
				{ID: 10}, {ID: 11}, {ID: 12},
			}},
		}

		count, err := svc.CountForSpace(t.Context(), space, true, &types.PullReqFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(9), count)
		require.Equal(t, []int64{10, 11, 12}, prs.gotSpace)
	})

	t.Run("errors", func(t *testing.T) {
		boom := errors.New("boom")

		svc := &ListService{pullreqStore: &countPullReqStore{}, spaceStore: countSpaceStore{err: boom}}
		_, err := svc.CountForSpace(t.Context(), space, true, &types.PullReqFilter{})
		require.ErrorIs(t, err, boom)

		svc = &ListService{pullreqStore: &countPullReqStore{err: boom}, spaceStore: countSpaceStore{}}
		_, err = svc.CountForSpace(t.Context(), space, false, &types.PullReqFilter{})
		require.ErrorIs(t, err, boom)
	})
}
