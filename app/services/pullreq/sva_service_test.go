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
	"sync/atomic"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func svaNewEventSystem(t *testing.T) *events.System {
	t.Helper()

	system, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 1000,
	}, nil, events.NewNoopCollector())
	require.NoError(t, err)

	return system
}

func svaNewService(t *testing.T, bus pubsub.PubSub) *Service {
	t.Helper()

	system := svaNewEventSystem(t)

	gitFactory, err := gitevents.NewReaderFactory(system)
	require.NoError(t, err)

	pullReqFactory, err := pullreqevents.NewReaderFactory(system)
	require.NoError(t, err)

	reporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	config := &types.Config{InstanceID: "instance"}
	config.PullReq.GitEventsConcurrency = 1
	config.PullReq.FileViewedConcurrency = 1
	config.PullReq.MergeabilityConcurrency = 1
	config.PullReq.CodeCommentsConcurrency = 1

	svc, err := ProvideService(
		t.Context(), config, gitFactory, pullReqFactory, reporter, nil, nil, nil, svaFixedRepoFinder(nil),
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, bus, nil, nil,
	)
	require.NoError(t, err)

	return svc
}

// svaCapturingBus records the subscription handlers instead of delivering messages.
type svaCapturingBus struct {
	pubsub.PubSub

	handlers map[string]func(payload []byte) error
}

func (b *svaCapturingBus) Subscribe(
	_ context.Context,
	topic string,
	handler func(payload []byte) error,
	_ ...pubsub.SubscribeOption,
) pubsub.Consumer {
	b.handlers[topic] = handler
	return nil
}

func TestSvaNewServiceWiresMergeabilityCancellation(t *testing.T) {
	t.Parallel()

	bus := &svaCapturingBus{handlers: map[string]func([]byte) error{}}
	svc := svaNewService(t, bus)
	require.NotNil(t, svc.cancelMergeability)

	handler := bus.handlers[cancelMergeCheckKey]
	require.NotNil(t, handler, "the service subscribes to cancellations of mergeability checks")

	var cancelled, unrelated atomic.Bool
	svc.cancelMergeability["old-sha"] = func() { cancelled.Store(true) }
	svc.cancelMergeability["other-sha"] = func() { unrelated.Store(true) }

	// An empty payload cancels nothing.
	require.NoError(t, handler(nil))
	require.False(t, cancelled.Load())

	// An unknown SHA is a no-op.
	require.NoError(t, handler([]byte("unknown-sha")))
	require.False(t, cancelled.Load())

	require.NoError(t, handler([]byte("old-sha")))
	require.True(t, cancelled.Load())
	require.False(t, unrelated.Load())
	require.NotContains(t, svc.cancelMergeability, "old-sha")
	require.Contains(t, svc.cancelMergeability, "other-sha")
}

func TestSvaNewWithReviewerStores(t *testing.T) {
	t.Parallel()

	svc := NewWithReviewerStores(&svaPullReqStore{}, nil, nil)
	require.NotNil(t, svc)
	require.NotNil(t, svc.pullreqStore)
}

func TestSvaProvideListService(t *testing.T) {
	t.Parallel()

	svc := ProvideListService(nil, nil, nil, nil, &svaPullReqStore{}, nil, svaFixedRepoFinder(nil), nil, nil)
	require.NotNil(t, svc)
	require.NotNil(t, svc.pullreqStore)
}
