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

package branch

import (
	"context"
	"errors"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

const svbSHA = "1111111111111111111111111111111111111111"

type svbBranchStore struct {
	store.BranchStore

	upserted *types.BranchTable
	repoID   int64
	deleted  string
	lastPR   *int64
	lastName string
	called   bool
	err      error
}

func (f *svbBranchStore) Upsert(_ context.Context, repoID int64, b *types.BranchTable) error {
	f.repoID, f.upserted = repoID, b
	return f.err
}

func (f *svbBranchStore) Delete(_ context.Context, repoID int64, name string) error {
	f.repoID, f.deleted = repoID, name
	return f.err
}

func (f *svbBranchStore) UpdateLastPR(_ context.Context, repoID int64, name string, prID *int64) error {
	f.called, f.repoID, f.lastName, f.lastPR = true, repoID, name, prID
	return f.err
}

func svbI64(v int64) *int64 { return &v }

func TestSvbConfigPrepare(t *testing.T) {
	var nilCfg *Config
	require.Error(t, nilCfg.Prepare())
	require.Error(t, (&Config{Concurrency: 1}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r"}).Prepare())
	require.Error(t, (&Config{EventReaderName: "r", Concurrency: 1, MaxRetries: -1}).Prepare())
	require.NoError(t, (&Config{EventReaderName: "r", Concurrency: 1}).Prepare())
}

func TestSvbExtractBranchName(t *testing.T) {
	require.Equal(t, "main", ExtractBranchName("refs/heads/main"))
	require.Equal(t, "a/b", ExtractBranchName("refs/heads/a/b"))
	require.Equal(t, "other", ExtractBranchName("other"))
}

func TestSvbBranchHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("created", func(t *testing.T) {
		bs := &svbBranchStore{}
		svc := &Service{branchStore: bs}
		err := svc.handleEventBranchCreated(ctx, &events.Event[*gitevents.BranchCreatedPayload]{
			Payload: &gitevents.BranchCreatedPayload{RepoID: 4, PrincipalID: 8, Ref: "refs/heads/feat", SHA: svbSHA},
		})
		require.NoError(t, err)
		require.Equal(t, int64(4), bs.repoID)
		require.Equal(t, "feat", bs.upserted.Name)
		require.Equal(t, svbSHA, bs.upserted.SHA.String())
		require.Equal(t, int64(8), bs.upserted.CreatedBy)
		require.Equal(t, int64(8), bs.upserted.UpdatedBy)
		require.NotZero(t, bs.upserted.Created)

		err = svc.handleEventBranchCreated(ctx, &events.Event[*gitevents.BranchCreatedPayload]{
			Payload: &gitevents.BranchCreatedPayload{Ref: "refs/heads/x", SHA: "zz"},
		})
		require.ErrorContains(t, err, "sha format invalid")

		bs.err = errSvbBoom
		err = svc.handleEventBranchCreated(ctx, &events.Event[*gitevents.BranchCreatedPayload]{
			Payload: &gitevents.BranchCreatedPayload{Ref: "refs/heads/x", SHA: svbSHA},
		})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("updated", func(t *testing.T) {
		bs := &svbBranchStore{}
		svc := &Service{branchStore: bs}
		err := svc.handleEventBranchUpdated(ctx, &events.Event[*gitevents.BranchUpdatedPayload]{
			Payload: &gitevents.BranchUpdatedPayload{RepoID: 4, Ref: "refs/heads/feat", NewSHA: svbSHA},
		})
		require.NoError(t, err)
		require.Equal(t, "feat", bs.upserted.Name)

		err = svc.handleEventBranchUpdated(ctx, &events.Event[*gitevents.BranchUpdatedPayload]{
			Payload: &gitevents.BranchUpdatedPayload{NewSHA: "zz"},
		})
		require.ErrorContains(t, err, "sha format invalid")

		bs.err = errSvbBoom
		err = svc.handleEventBranchUpdated(ctx, &events.Event[*gitevents.BranchUpdatedPayload]{
			Payload: &gitevents.BranchUpdatedPayload{NewSHA: svbSHA},
		})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("deleted", func(t *testing.T) {
		bs := &svbBranchStore{}
		svc := &Service{branchStore: bs}
		err := svc.handleEventBranchDeleted(ctx, &events.Event[*gitevents.BranchDeletedPayload]{
			Payload: &gitevents.BranchDeletedPayload{RepoID: 4, Ref: "refs/heads/feat"},
		})
		require.NoError(t, err)
		require.Equal(t, "feat", bs.deleted)
		require.Equal(t, int64(4), bs.repoID)

		bs.err = errSvbBoom
		err = svc.handleEventBranchDeleted(ctx, &events.Event[*gitevents.BranchDeletedPayload]{
			Payload: &gitevents.BranchDeletedPayload{Ref: "refs/heads/feat"},
		})
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbPullReqHandlers(t *testing.T) {
	ctx := context.Background()
	base := func(sourceRepo *int64) pullreqevents.Base {
		return pullreqevents.Base{PullReqID: 77, SourceRepoID: sourceRepo}
	}

	type handler func(ctx context.Context, svc *Service, sourceRepo *int64, nilPayload bool) error
	tests := []struct {
		name       string
		call       handler
		wantPRSet  bool
		wantBranch string
	}{
		{
			name: "created",
			call: func(ctx context.Context, svc *Service, repo *int64, nilPayload bool) error {
				ev := &events.Event[*pullreqevents.CreatedPayload]{}
				if !nilPayload {
					ev.Payload = &pullreqevents.CreatedPayload{Base: base(repo), SourceBranch: "feat"}
				}
				return svc.handleEventPullReqCreated(ctx, ev)
			},
			wantPRSet: true,
		},
		{
			name: "reopened",
			call: func(ctx context.Context, svc *Service, repo *int64, nilPayload bool) error {
				ev := &events.Event[*pullreqevents.ReopenedPayload]{}
				if !nilPayload {
					ev.Payload = &pullreqevents.ReopenedPayload{Base: base(repo), SourceBranch: "feat"}
				}
				return svc.handleEventPullReqReopened(ctx, ev)
			},
			wantPRSet: true,
		},
		{
			name: "closed",
			call: func(ctx context.Context, svc *Service, repo *int64, nilPayload bool) error {
				ev := &events.Event[*pullreqevents.ClosedPayload]{}
				if !nilPayload {
					ev.Payload = &pullreqevents.ClosedPayload{Base: base(repo), SourceBranch: "feat"}
				}
				return svc.handleEventPullReqClosed(ctx, ev)
			},
			wantPRSet: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bs := &svbBranchStore{}
			svc := &Service{branchStore: bs}

			require.NoError(t, tc.call(ctx, svc, svbI64(5), false))
			require.Equal(t, int64(5), bs.repoID)
			require.Equal(t, "feat", bs.lastName)
			if tc.wantPRSet {
				require.Equal(t, int64(77), *bs.lastPR)
			} else {
				require.Nil(t, bs.lastPR)
			}

			require.Error(t, tc.call(ctx, svc, svbI64(5), true), "nil payload")

			bs.called = false
			err := tc.call(ctx, svc, nil, false)
			require.Error(t, err, "missing source repo is discarded")
			require.False(t, bs.called)

			bs.err = errSvbBoom
			require.ErrorIs(t, tc.call(ctx, svc, svbI64(5), false), errSvbBoom)
		})
	}
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

func TestSvbNew(t *testing.T) {
	ctx := context.Background()
	cfg := Config{EventReaderName: "reader", Concurrency: 1}

	build := func(t *testing.T, system *events.System, cfg Config) (*Service, error) {
		t.Helper()
		gitFactory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(system)
		require.NoError(t, err)
		return New(ctx, cfg, nil, gitFactory, prFactory, nil)
	}

	consumer := &svbConsumer{}
	svc, err := build(t, svbSystem(t, consumer, 2), cfg)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 3+3, consumer.registered)

	_, err = build(t, svbSystem(t, &svbConsumer{}, 2), Config{})
	require.Error(t, err)

	_, err = build(t, svbSystem(t, &svbConsumer{}, 0), cfg)
	require.ErrorContains(t, err, "git events reader")

	_, err = build(t, svbSystem(t, &svbConsumer{}, 1), cfg)
	require.ErrorContains(t, err, "pullreq events reader")
}
