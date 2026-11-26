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

package instrument

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: 77, UID: "sys", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

type sv5RepoStore struct {
	store.RepoStore

	counts []types.RepositoryCount
	err    error
}

func (s sv5RepoStore) CountByRootSpaces(context.Context) ([]types.RepositoryCount, error) {
	return s.counts, s.err
}

type sv5Service struct {
	Service

	err    error
	events []Event
}

func (s *sv5Service) Track(_ context.Context, e Event) error {
	s.events = append(s.events, e)
	return s.err
}

func TestSv5RepositoryCountHandle(t *testing.T) {
	require.True(t, sv5Booted)
	ctx := context.Background()
	counts := []types.RepositoryCount{{SpaceID: 1, SpaceUID: "root1", Total: 3}, {SpaceID: 2, SpaceUID: "root2", Total: 5}}

	t.Run("disabled", func(t *testing.T) {
		c := &RepositoryCount{}
		_, err := c.Handle(ctx, "", nil)
		require.ErrorContains(t, err, "disabled")
	})

	t.Run("missing repo store", func(t *testing.T) {
		c := &RepositoryCount{enabled: true}
		_, err := c.Handle(ctx, "", nil)
		require.ErrorContains(t, err, "repository store not initialized")
	})

	t.Run("count failure", func(t *testing.T) {
		c := &RepositoryCount{enabled: true, repoStore: sv5RepoStore{err: errSv5Boom}}
		_, err := c.Handle(ctx, "", nil)
		require.ErrorIs(t, err, errSv5Boom)
	})

	t.Run("missing service", func(t *testing.T) {
		c := &RepositoryCount{enabled: true, repoStore: sv5RepoStore{counts: counts}}
		_, err := c.Handle(ctx, "", nil)
		require.ErrorContains(t, err, "service not initialized")
	})

	t.Run("tracks one event per root space", func(t *testing.T) {
		svc := &sv5Service{}
		c := &RepositoryCount{enabled: true, svc: svc, repoStore: sv5RepoStore{counts: counts}}

		res, err := c.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Empty(t, res)
		require.Len(t, svc.events, 2)
		require.Equal(t, EventTypeRepositoryCount, svc.events[0].Type)
		require.Equal(t, "root1", svc.events[0].GroupID)
		require.Equal(t, int64(77), svc.events[0].Principal.ID)
		require.Equal(t, 5, svc.events[1].Properties[PropertyRepositories])
	})

	t.Run("tracking failures are tolerated", func(t *testing.T) {
		svc := &sv5Service{err: errSv5Boom}
		c := &RepositoryCount{enabled: true, svc: svc, repoStore: sv5RepoStore{counts: counts}}
		_, err := c.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Len(t, svc.events, 2)
	})
}
