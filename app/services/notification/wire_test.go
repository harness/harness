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

package notification

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/idempotency"
	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

type countingMailer struct{ sent int }

func (m *countingMailer) Send(context.Context, mailer.Payload) error {
	m.sent++
	return nil
}

// Outside of redis mode the mails are deduplicated in memory and redis must not be touched.
func TestProvideMailClient_InMemoryModeDeduplicates(t *testing.T) {
	t.Parallel()

	config := &types.Config{}
	config.Events.Mode = events.ModeInMemory

	inner := &countingMailer{}
	client, ok := ProvideMailClient(config, inner, nil).(MailClient)
	require.True(t, ok)

	ctx := mailer.WithEventKey(context.Background(), "event-1")
	payload := mailer.Payload{ToRecipients: []string{"a@x.io"}, Subject: "s", Body: "b"}

	require.NoError(t, client.Mailer.Send(ctx, payload))
	require.NoError(t, client.Mailer.Send(ctx, payload))
	require.Equal(t, 1, inner.sent)
}

// In redis mode the mails are deduplicated through the shared redis client, which is only used once a mail is sent.
func TestProvideMailClient_RedisModeUsesRedisStore(t *testing.T) {
	t.Parallel()

	config := &types.Config{}
	config.Events.Mode = events.ModeRedis

	inner := &countingMailer{}
	client, ok := ProvideMailClient(config, inner, redis.NewClient(&redis.Options{})).(MailClient)
	require.True(t, ok)

	// Without an event key nothing is deduplicated and redis isn't touched.
	payload := mailer.Payload{ToRecipients: []string{"a@x.io"}, Subject: "s", Body: "b"}
	require.NoError(t, client.Mailer.Send(context.Background(), payload))
	require.NoError(t, client.Mailer.Send(context.Background(), payload))
	require.Equal(t, 2, inner.sent)
}

// The handlers have to hand the event ID to the mailer, or redelivered events are never deduplicated.
func TestWithEventKey_DeduplicatesMailsOfRedeliveredEvent(t *testing.T) {
	t.Parallel()

	inner := &countingMailer{}
	m := mailer.NewDedupMailer(inner, idempotency.NewInMemory(), time.Hour)
	payload := mailer.Payload{ToRecipients: []string{"a@x.io"}, Subject: "s", Body: "b"}

	handler := withEventKey(func(ctx context.Context, _ *events.Event[struct{}]) error {
		return m.Send(ctx, payload)
	})

	require.NoError(t, handler(context.Background(), &events.Event[struct{}]{ID: "1-0"}))
	require.NoError(t, handler(context.Background(), &events.Event[struct{}]{ID: "1-0"}))
	require.Equal(t, 1, inner.sent)

	require.NoError(t, handler(context.Background(), &events.Event[struct{}]{ID: "2-0"}))
	require.Equal(t, 2, inner.sent)
}
