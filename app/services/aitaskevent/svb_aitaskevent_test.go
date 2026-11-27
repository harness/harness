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

package aitaskevent

import (
	"context"
	"errors"
	"testing"

	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/app/services/gitspaceevent"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbOrchestrator struct {
	err    error
	called int
}

func (o *svbOrchestrator) TriggerAITask(context.Context, types.AITask, types.GitspaceConfig) error {
	o.called++
	return o.err
}

type svbAITaskStore struct {
	store.AITaskStore

	task *types.AITask
	err  error
}

func (f *svbAITaskStore) FindByIdentifier(context.Context, int64, string) (*types.AITask, error) {
	return f.task, f.err
}

func TestSvbHandleStartEvent(t *testing.T) {
	ctx := context.Background()
	logr := zerolog.Nop()
	running := types.GitspaceConfig{State: enum.GitspaceStateRunning}
	stopped := types.GitspaceConfig{State: enum.GitspaceStateStopped}

	tests := []struct {
		name       string
		state      enum.AITaskState
		cfg        types.GitspaceConfig
		orchErr    error
		wantErr    string
		wantCalled int
	}{
		{"uninitialized triggers", enum.AITaskStateUninitialized, running, nil, "", 1},
		{"running re-triggers", enum.AITaskStateRunning, running, nil, "", 1},
		{"error re-triggers", enum.AITaskStateError, running, nil, "", 1},
		{"completed is a no-op", enum.AITaskStateCompleted, running, nil, "", 0},
		{"invalid state", enum.AITaskState("bogus"), running, nil, "invalid state", 0},
		{"gitspace not running", enum.AITaskStateUninitialized, stopped, nil, "gitspace is not running", 0},
		{"orchestrator error", enum.AITaskStateUninitialized, running, errSvbBoom, "boom", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orch := &svbOrchestrator{err: tc.orchErr}
			svc := &Service{orchestrator: orch}
			err := svc.handleStartEvent(ctx, types.AITask{Identifier: "t", State: tc.state}, tc.cfg, logr)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			require.Equal(t, tc.wantCalled, orch.called)
		})
	}
}

func TestSvbHandleStopEvent(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.handleStopEvent(context.Background(), &aitaskevents.AITaskEventPayload{}))
}

func TestSvbFetchWithRetry(t *testing.T) {
	ctx := context.Background()

	task := &types.AITask{Identifier: "t1"}
	svc := &Service{aiTaskStore: &svbAITaskStore{task: task}}
	got, err := svc.fetchWithRetry(ctx, "t1", 1)
	require.NoError(t, err)
	require.Same(t, task, got)

	// Errors other than "not found" are returned right away without retrying.
	svc = &Service{aiTaskStore: &svbAITaskStore{err: errSvbBoom}}
	got, err = svc.fetchWithRetry(ctx, "t1", 1)
	require.ErrorIs(t, err, errSvbBoom)
	require.Nil(t, got)
	require.NotErrorIs(t, err, gitnessstore.ErrResourceNotFound)
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

func svbFactory(t *testing.T, consumer *svbConsumer, factoryErr error) *events.ReaderFactory[*aitaskevents.Reader] {
	t.Helper()
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			if factoryErr != nil {
				return nil, factoryErr
			}
			return consumer, nil
		},
		svbProducer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	factory, err := aitaskevents.NewReaderFactory(system)
	require.NoError(t, err)
	return factory
}

func TestSvbNewService(t *testing.T) {
	ctx := context.Background()
	cfg := &gitspaceevent.Config{EventReaderName: "reader", Concurrency: 1, TimeoutInMins: 1}

	consumer := &svbConsumer{}
	svc, err := NewService(ctx, cfg, svbFactory(t, consumer, nil), &svbOrchestrator{}, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 1, consumer.registered)

	_, err = NewService(ctx, &gitspaceevent.Config{}, svbFactory(t, consumer, nil), &svbOrchestrator{}, nil, nil)
	require.ErrorContains(t, err, "config is invalid")

	_, err = NewService(ctx, cfg, svbFactory(t, nil, errSvbBoom), &svbOrchestrator{}, nil, nil)
	require.ErrorContains(t, err, "failed to launch ai task event reader")
}
