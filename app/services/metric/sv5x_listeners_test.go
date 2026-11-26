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

package metric

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	repoevents "github.com/harness/gitness/app/events/repo"
	ruleevents "github.com/harness/gitness/app/events/rule"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/events"
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

type sv5xSubmitter struct{ Submitter }

func TestSv5xRegisterEventListeners(t *testing.T) {
	ctx := context.Background()
	config := &types.Config{InstanceID: "instance"}
	config.Metric.UserEventsConcurrency = 1
	config.Metric.RepoEventsConcurrency = 1
	config.Metric.PullReqEventsConcurrency = 1
	config.Metric.RuleEventsConcurrency = 1

	run := func(t *testing.T, okLaunches int, submitter Submitter) (*sv5xConsumer, error) {
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

		userFactory, err := userevents.NewReaderFactory(system)
		require.NoError(t, err)
		repoFactory, err := repoevents.NewReaderFactory(system)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(system)
		require.NoError(t, err)
		ruleFactory, err := ruleevents.NewReaderFactory(system)
		require.NoError(t, err)

		err = registerEventListeners(
			ctx, config, nil, nil, nil, userFactory, repoFactory, prFactory, ruleFactory,
			refcache.SpaceFinder{}, refcache.RepoFinder{}, nil, submitter,
		)
		return consumer, err
	}

	t.Run("no submitter registers nothing", func(t *testing.T) {
		consumer, err := run(t, 4, nil)
		require.NoError(t, err)
		require.Zero(t, consumer.registered)
	})

	t.Run("registers all handlers", func(t *testing.T) {
		consumer, err := run(t, 4, sv5xSubmitter{})
		require.NoError(t, err)
		// 3 user + 3 repo + 5 pull request + 1 rule events.
		require.Equal(t, 3+3+5+1, consumer.registered)
	})

	for launched := range 4 {
		t.Run("launch failure", func(t *testing.T) {
			_, err := run(t, launched, sv5xSubmitter{})
			require.ErrorIs(t, err, errSv5xBoom)
		})
	}
}
