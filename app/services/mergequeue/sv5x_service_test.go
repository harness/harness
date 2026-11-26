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

package mergequeue

import (
	"context"
	"errors"
	"testing"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSv5xBoom = errors.New("boom")

type sv5xConsumer struct{ registered int }

func (c *sv5xConsumer) Register(string, stream.HandlerFunc, ...stream.HandlerOption) error {
	c.registered++
	return nil
}

func (c *sv5xConsumer) Configure(...stream.ConsumerOption) {}

func (c *sv5xConsumer) Start(context.Context) error { return nil }

func (c *sv5xConsumer) Errors() <-chan error { return make(chan error) }

func (c *sv5xConsumer) Infos() <-chan string { return make(chan string) }

type sv5xProducer struct{}

func (sv5xProducer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

type sv5xJobStore struct {
	job.Store

	err error
	uid string
}

func (s *sv5xJobStore) Upsert(_ context.Context, j *job.Job) error {
	s.uid = j.UID
	return s.err
}

func TestSv5xNewService(t *testing.T) {
	ctx := context.Background()
	config := &types.Config{InstanceID: "instance"}
	config.MergeQueue.CheckEventsConcurrency = 1
	config.MergeQueue.QueueEventsConcurrency = 1

	build := func(
		t *testing.T, okLaunches int, store *sv5xJobStore, executor *job.Executor,
	) (*Service, *sv5xConsumer, error) {
		t.Helper()
		consumer := &sv5xConsumer{}
		calls := 0
		system, err := events.NewSystem(
			func(string, string) (events.StreamConsumer, error) {
				calls++
				if calls > okLaunches {
					return nil, errSv5xBoom
				}
				return consumer, nil
			},
			sv5xProducer{},
			events.NewNoopCollector(),
		)
		require.NoError(t, err)
		checkFactory, err := checkevents.NewReaderFactory(system)
		require.NoError(t, err)
		mqFactory, err := mergequeueevents.NewReaderFactory(system)
		require.NoError(t, err)
		scheduler, err := job.NewScheduler(store, nil, nil, nil, "inst", 1, time.Hour)
		require.NoError(t, err)
		if executor == nil {
			executor = job.NewExecutor(store, nil)
		}

		svc, err := NewService(
			ctx, config, nil, nil, nil, nil, checkFactory, mqFactory, refcache.RepoFinder{}, nil, nil, nil, nil, nil, nil, nil,
			nil, nil, nil, scheduler, executor,
		)
		return svc, consumer, err
	}

	t.Run("registers handlers and the overdue checks job", func(t *testing.T) {
		store := &sv5xJobStore{}
		svc, consumer, err := build(t, 2, store, nil)
		require.NoError(t, err)
		require.NotNil(t, svc)
		require.Equal(t, 1+2, consumer.registered, "check reported plus updated and branch deleted")
		require.Equal(t, jobTypeOverdueChecks, store.uid)
	})

	t.Run("check reader launch failure", func(t *testing.T) {
		_, _, err := build(t, 0, &sv5xJobStore{}, nil)
		require.ErrorIs(t, err, errSv5xBoom)
	})

	t.Run("merge queue reader launch failure", func(t *testing.T) {
		_, _, err := build(t, 1, &sv5xJobStore{}, nil)
		require.ErrorIs(t, err, errSv5xBoom)
	})

	t.Run("job registered twice", func(t *testing.T) {
		store := &sv5xJobStore{}
		executor := job.NewExecutor(store, nil)
		require.NoError(t, executor.Register(jobTypeOverdueChecks, &jobOverdueChecks{}))
		_, _, err := build(t, 2, store, executor)
		require.ErrorContains(t, err, "register merge queue overdue checks job")
	})

	t.Run("job scheduling failure", func(t *testing.T) {
		_, _, err := build(t, 2, &sv5xJobStore{err: errSv5xBoom}, nil)
		require.ErrorIs(t, err, errSv5xBoom)
		require.ErrorContains(t, err, "schedule merge queue overdue checks job")
	})
}
