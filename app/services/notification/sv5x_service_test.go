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

package notification

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"

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

func sv5xFactory(
	t *testing.T, consumer *sv5xConsumer, launchErr error,
) *events.ReaderFactory[*pullreqevents.Reader] {
	t.Helper()
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			if launchErr != nil {
				return nil, launchErr
			}
			return consumer, nil
		},
		sv5xProducer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	factory, err := pullreqevents.NewReaderFactory(system)
	require.NoError(t, err)
	return factory
}

func TestSv5xNewService(t *testing.T) {
	ctx := context.Background()
	cfg := Config{EventReaderName: "reader", Concurrency: 1, MaxRetries: 1}

	consumer := &sv5xConsumer{}
	svc, err := NewService(
		ctx, cfg, nil, sv5xFactory(t, consumer, nil), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, cfg, svc.config)
	require.Equal(t, 9, consumer.registered, "created, reviewers, comments, branch, review and 3 state events")

	_, err = NewService(
		ctx, cfg, nil, sv5xFactory(t, nil, errSv5xBoom), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	require.ErrorIs(t, err, errSv5xBoom)
	require.ErrorContains(t, err, eventReaderGroupName)
}

func TestSv5xLoadTemplates(t *testing.T) {
	require.NoError(t, LoadTemplates())
	require.Contains(t, htmlTemplates, TemplateReviewerAdded)
	require.Contains(t, htmlTemplates, TemplatePullReqStateChanged)
	require.Len(t, htmlTemplates, 8)
}
