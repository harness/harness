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

package repoactivity

import (
	"context"
	"errors"
	"testing"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5Store struct {
	store.RepoActivityStore

	err        error
	activities []*types.RepoActivity
}

func (s *sv5Store) Create(_ context.Context, a *types.RepoActivity) error {
	s.activities = append(s.activities, a)
	return s.err
}

func sv5Event[T any](payload T) *events.Event[T] {
	return &events.Event[T]{ID: "evt-1", Timestamp: time.UnixMilli(1234), Payload: payload}
}

func TestSv5ExtractBranchName(t *testing.T) {
	require.Equal(t, "feature/x", extractBranchName("refs/heads/feature/x"))
	require.Equal(t, "main", extractBranchName("main"))
}

func TestSv5BranchHandlers(t *testing.T) {
	ctx := context.Background()
	st := &sv5Store{}
	svc := &Service{repoActivityStore: st}

	require.NoError(t, svc.handleEventBranchCreated(ctx, sv5Event(&gitevents.BranchCreatedPayload{
		RepoID: 1, PrincipalID: 2, Ref: "refs/heads/dev", SHA: "aaaa",
	})))
	require.NoError(t, svc.handleEventBranchUpdated(ctx, sv5Event(&gitevents.BranchUpdatedPayload{
		RepoID: 1, PrincipalID: 2, Ref: "refs/heads/dev", OldSHA: "aaaa", NewSHA: "bbbb", Forced: true,
	})))
	require.NoError(t, svc.handleEventBranchDeleted(ctx, sv5Event(&gitevents.BranchDeletedPayload{
		RepoID: 1, PrincipalID: 2, Ref: "refs/heads/dev", SHA: "bbbb",
	})))

	require.Len(t, st.activities, 3)
	for _, a := range st.activities {
		require.Equal(t, int64(1), a.RepoID)
		require.Equal(t, int64(2), a.PrincipalID)
		require.Equal(t, "evt-1", a.Key)
		require.Equal(t, int64(1234), a.Timestamp)
	}

	require.Equal(t, enum.RepoActivityTypeBranchCreated, st.activities[0].Type)
	require.Equal(t, &types.RepoActivityPayloadBranchCreated{Name: "dev", New: "aaaa"}, st.activities[0].Payload)
	require.Equal(t, enum.RepoActivityTypeBranchUpdated, st.activities[1].Type)
	require.Equal(t,
		&types.RepoActivityPayloadBranchUpdated{Name: "dev", Old: "aaaa", New: "bbbb", Forced: true},
		st.activities[1].Payload)
	require.Equal(t, enum.RepoActivityTypeBranchDeleted, st.activities[2].Type)
	require.Equal(t, &types.RepoActivityPayloadBranchDeleted{Name: "dev", Old: "bbbb"}, st.activities[2].Payload)
}

func TestSv5BranchHandlersStoreFailure(t *testing.T) {
	ctx := context.Background()
	svc := &Service{repoActivityStore: &sv5Store{err: errSv5Boom}}

	err := svc.handleEventBranchCreated(ctx, sv5Event(&gitevents.BranchCreatedPayload{Ref: "refs/heads/a"}))
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "branch creation")

	err = svc.handleEventBranchUpdated(ctx, sv5Event(&gitevents.BranchUpdatedPayload{Ref: "refs/heads/a"}))
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "branch update")

	err = svc.handleEventBranchDeleted(ctx, sv5Event(&gitevents.BranchDeletedPayload{Ref: "refs/heads/a"}))
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "branch deletion")
}

func TestSv5ConfigPrepare(t *testing.T) {
	var nilConfig *Config
	require.ErrorContains(t, nilConfig.Prepare(), "config is required")
	require.ErrorContains(t, (&Config{Concurrency: 1}).Prepare(), "EventReaderName")
	require.ErrorContains(t, (&Config{EventReaderName: "r"}).Prepare(), "Concurrency")
	require.ErrorContains(t, (&Config{EventReaderName: "r", Concurrency: 1, MaxRetries: -1}).Prepare(), "MaxRetries")
	require.NoError(t, (&Config{EventReaderName: "r", Concurrency: 1}).Prepare())
}

type sv5Consumer struct{ registered int }

func (c *sv5Consumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *sv5Consumer) Configure(...stream.ConsumerOption) {}

func (c *sv5Consumer) Start(context.Context) error { return nil }

func (c *sv5Consumer) Errors() <-chan error { return make(chan error) }

func (c *sv5Consumer) Infos() <-chan string { return make(chan string) }

type sv5Producer struct{}

func (sv5Producer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func TestSv5New(t *testing.T) {
	ctx := context.Background()
	cfg := Config{EventReaderName: "reader", Concurrency: 1}

	build := func(t *testing.T, consumerErr error, cfg Config) (*Service, *sv5Consumer, error) {
		t.Helper()
		consumer := &sv5Consumer{}
		system, err := events.NewSystem(
			func(string, string) (events.StreamConsumer, error) {
				if consumerErr != nil {
					return nil, consumerErr
				}
				return consumer, nil
			},
			sv5Producer{},
			events.NewNoopCollector(),
			false,
		)
		require.NoError(t, err)
		factory, err := gitevents.NewReaderFactory(system)
		require.NoError(t, err)
		svc, err := New(ctx, cfg, &sv5Store{}, factory)
		return svc, consumer, err
	}

	svc, consumer, err := build(t, nil, cfg)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 3, consumer.registered)

	_, _, err = build(t, nil, Config{})
	require.ErrorContains(t, err, "repo activity service config is invalid")

	_, _, err = build(t, errSv5Boom, cfg)
	require.ErrorIs(t, err, errSv5Boom)
	require.ErrorContains(t, err, "git events reader")
}
