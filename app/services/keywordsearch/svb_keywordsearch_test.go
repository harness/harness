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

package keywordsearch

import (
	"context"
	"errors"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbRepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (f *svbRepoStore) Find(context.Context, int64) (*types.Repository, error) { return f.repo, f.err }

type svbIndexer struct {
	indexed []*types.Repository
	err     error
}

func (i *svbIndexer) Index(_ context.Context, repo *types.Repository) error {
	i.indexed = append(i.indexed, repo)
	return i.err
}

func TestSvbConfigPrepare(t *testing.T) {
	var nilCfg *Config
	require.Error(t, nilCfg.Prepare())
	require.Error(t, (&Config{Concurrency: 1}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r"}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r", Concurrency: 1, MaxRetries: -1}).Prepare())
	require.NoError(t, (&Config{EventReaderName: "r", Concurrency: 1}).Prepare())
}

func TestSvbGetBranchFromRef(t *testing.T) {
	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{"refs/heads/main", "main", false},
		{"refs/heads/feature/x", "feature/x", false},
		{"refs/tags/v1", "", true},
		{"main", "", true},
		{"refs/heads/", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			got, err := getBranchFromRef(tc.ref)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSvbIndexRepo(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{ID: 3, DefaultBranch: "main"}

	t.Run("indexes the default branch", func(t *testing.T) {
		idx := &svbIndexer{}
		svc := &Service{repoStore: &svbRepoStore{repo: repo}, indexer: idx}
		require.NoError(t, svc.indexRepo(ctx, 3, "refs/heads/main"))
		require.Equal(t, []*types.Repository{repo}, idx.indexed)
	})

	t.Run("ignores other branches", func(t *testing.T) {
		idx := &svbIndexer{}
		svc := &Service{repoStore: &svbRepoStore{repo: repo}, indexer: idx}
		require.NoError(t, svc.indexRepo(ctx, 3, "refs/heads/dev"))
		require.Empty(t, idx.indexed)
	})

	t.Run("invalid ref is discarded", func(t *testing.T) {
		svc := &Service{repoStore: &svbRepoStore{repo: repo}, indexer: &svbIndexer{}}
		require.Error(t, svc.indexRepo(ctx, 3, "refs/tags/v1"))
	})

	t.Run("failures", func(t *testing.T) {
		svc := &Service{repoStore: &svbRepoStore{err: errSvbBoom}, indexer: &svbIndexer{}}
		require.ErrorIs(t, svc.indexRepo(ctx, 3, "refs/heads/main"), errSvbBoom)

		svc = &Service{repoStore: &svbRepoStore{repo: repo}, indexer: &svbIndexer{err: errSvbBoom}}
		require.ErrorIs(t, svc.indexRepo(ctx, 3, "refs/heads/main"), errSvbBoom)
	})
}

func TestSvbEventHandlers(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{ID: 3, DefaultBranch: "main"}
	idx := &svbIndexer{}
	svc := &Service{repoStore: &svbRepoStore{repo: repo}, indexer: idx}

	require.NoError(t, svc.handleEventBranchCreated(ctx, &events.Event[*gitevents.BranchCreatedPayload]{
		Payload: &gitevents.BranchCreatedPayload{RepoID: 3, Ref: "refs/heads/main"},
	}))
	require.NoError(t, svc.handleEventBranchUpdated(ctx, &events.Event[*gitevents.BranchUpdatedPayload]{
		Payload: &gitevents.BranchUpdatedPayload{RepoID: 3, Ref: "refs/heads/main"},
	}))
	require.Len(t, idx.indexed, 2)

	update := &events.Event[*repoevents.DefaultBranchUpdatedPayload]{
		Payload: &repoevents.DefaultBranchUpdatedPayload{Base: repoevents.Base{RepoID: 3}},
	}
	require.NoError(t, svc.handleUpdateDefaultBranch(ctx, update))
	require.Len(t, idx.indexed, 3)

	idx.err = errSvbBoom
	require.ErrorIs(t, svc.handleUpdateDefaultBranch(ctx, update), errSvbBoom)

	svc.repoStore = &svbRepoStore{err: errSvbBoom}
	require.ErrorIs(t, svc.handleUpdateDefaultBranch(ctx, update), errSvbBoom)
}

func TestSvbLocalIndexSearcher(t *testing.T) {
	s := NewLocalIndexSearcher()
	res, err := s.Search(context.Background(), []int64{1}, "q", false, 10)
	require.NoError(t, err)
	require.Equal(t, types.SearchResult{}, res)
	require.NoError(t, s.Index(context.Background(), &types.Repository{}))
}

type svbConsumer struct{ registered int }

func (c *svbConsumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *svbConsumer) Configure(...stream.ConsumerOption) {}

func (c *svbConsumer) Start(context.Context) error { return nil }

func (c *svbConsumer) Errors() <-chan error { return make(chan error) }

func (c *svbConsumer) Infos() <-chan string { return make(chan string) }

type svbProducer struct{}

func (svbProducer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func svbSystem(t *testing.T, consumer *svbConsumer, failAfter int) *events.System {
	t.Helper()
	calls := 0
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			calls++
			if calls > failAfter {
				return nil, errSvbBoom
			}
			return consumer, nil
		},
		svbProducer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	return system
}

func TestSvbNewService(t *testing.T) {
	ctx := context.Background()
	cfg := Config{EventReaderName: "reader", Concurrency: 1}

	build := func(t *testing.T, system *events.System, cfg Config) (*Service, error) {
		t.Helper()
		gitFactory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		repoFactory, err := repoevents.NewReaderFactory(system)
		require.NoError(t, err)
		return NewService(ctx, cfg, gitFactory, repoFactory, nil, &svbIndexer{})
	}

	consumer := &svbConsumer{}
	svc, err := build(t, svbSystem(t, consumer, 2), cfg)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 3, consumer.registered)

	_, err = build(t, svbSystem(t, &svbConsumer{}, 2), Config{})
	require.Error(t, err)

	_, err = build(t, svbSystem(t, &svbConsumer{}, 0), cfg)
	require.ErrorContains(t, err, "git event reader")

	_, err = build(t, svbSystem(t, &svbConsumer{}, 1), cfg)
	require.ErrorContains(t, err, "repo git group")
}
