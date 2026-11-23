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

package mailer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/idempotency"

	"github.com/stretchr/testify/require"
)

type mailerStub struct {
	calls  []Payload
	err    error
	onSend func()
}

func (s *mailerStub) Send(_ context.Context, payload Payload) error {
	s.calls = append(s.calls, payload)
	if s.onSend != nil {
		s.onSend()
	}
	return s.err
}

func newTestPayload(recipient string) Payload {
	return Payload{ToRecipients: []string{recipient}, Subject: "subject", Body: "body"}
}

func TestDedupMailer_SkipsRepeatedSendForSameEvent(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)
	ctx := WithEventKey(context.Background(), "event-1")

	require.NoError(t, m.Send(ctx, newTestPayload("a@example.com")))
	require.NoError(t, m.Send(ctx, newTestPayload("a@example.com")))

	require.Len(t, stub.calls, 1)
}

func TestDedupMailer_DistinctMailsOfSameEventAreAllSent(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)
	ctx := WithEventKey(context.Background(), "event-1")

	require.NoError(t, m.Send(ctx, newTestPayload("a@example.com")))
	require.NoError(t, m.Send(ctx, newTestPayload("b@example.com")))

	require.Len(t, stub.calls, 2)
}

func TestDedupMailer_SameMailForDifferentEventsIsSent(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)

	require.NoError(t, m.Send(WithEventKey(context.Background(), "event-1"), newTestPayload("a@example.com")))
	require.NoError(t, m.Send(WithEventKey(context.Background(), "event-2"), newTestPayload("a@example.com")))

	require.Len(t, stub.calls, 2)
}

func TestDedupMailer_FailedSendCanBeRetried(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{err: errors.New("smtp down")}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)
	ctx := WithEventKey(context.Background(), "event-1")

	require.Error(t, m.Send(ctx, newTestPayload("a@example.com")))

	stub.err = nil
	require.NoError(t, m.Send(ctx, newTestPayload("a@example.com")))

	require.Len(t, stub.calls, 2)
}

func TestDedupMailer_WithoutEventKeyAlwaysSends(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)

	require.NoError(t, m.Send(context.Background(), newTestPayload("a@example.com")))
	require.NoError(t, m.Send(context.Background(), newTestPayload("a@example.com")))

	require.Len(t, stub.calls, 2)
}

// ctxAwareStore fails Release on a canceled context, like a redis client does.
type ctxAwareStore struct {
	*idempotency.InMemoryStore
}

func (s ctxAwareStore) Release(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.InMemoryStore.Release(ctx, key)
}

func TestDedupMailer_FailedSendIsReleasedEvenIfContextIsCanceled(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{err: errors.New("smtp interrupted")}
	m := NewDedupMailer(stub, ctxAwareStore{idempotency.NewInMemory()}, time.Hour)

	ctx, cancel := context.WithCancel(WithEventKey(context.Background(), "event-1"))
	stub.onSend = cancel // shutdown happens while the mail is being sent

	require.Error(t, m.Send(ctx, newTestPayload("a@example.com")))

	stub.err, stub.onSend = nil, nil
	retryCtx := WithEventKey(context.Background(), "event-1")
	require.NoError(t, m.Send(retryCtx, newTestPayload("a@example.com")))

	require.Len(t, stub.calls, 2)
}

// ttlRecordingStore records the TTLs the mailer claims and commits with.
type ttlRecordingStore struct {
	*idempotency.InMemoryStore
	claimTTL, commitTTL time.Duration
}

func (s *ttlRecordingStore) Claim(
	ctx context.Context,
	key string,
	ttl time.Duration,
) (idempotency.State, error) {
	s.claimTTL = ttl
	return s.InMemoryStore.Claim(ctx, key, ttl)
}

func (s *ttlRecordingStore) Commit(ctx context.Context, key string, ttl time.Duration) error {
	s.commitTTL = ttl
	return s.InMemoryStore.Commit(ctx, key, ttl)
}

// The claim is short lived until the send succeeded.
func TestDedupMailer_ClaimIsShortUntilSendSucceeded(t *testing.T) {
	t.Parallel()

	store := &ttlRecordingStore{InMemoryStore: idempotency.NewInMemory()}
	m := NewDedupMailer(&mailerStub{}, store, time.Hour)

	require.NoError(t, m.Send(WithEventKey(context.Background(), "e1"), newTestPayload("a@x.io")))

	require.Equal(t, inFlightTTL, store.claimTTL)
	require.Equal(t, time.Hour, store.commitTTL)
}

func TestDedupMailer_FailedSendIsNotCommitted(t *testing.T) {
	t.Parallel()

	store := &ttlRecordingStore{InMemoryStore: idempotency.NewInMemory()}
	m := NewDedupMailer(&mailerStub{err: errors.New("smtp down")}, store, time.Hour)

	require.Error(t, m.Send(WithEventKey(context.Background(), "e1"), newTestPayload("a@x.io")))
	require.Zero(t, store.commitTTL)
}

// A mail another worker is still sending must fail instead of being reported as sent.
func TestDedupMailer_MailInFlightElsewhereFailsInsteadOfSkipping(t *testing.T) {
	t.Parallel()

	store := idempotency.NewInMemory()
	stub := &mailerStub{}
	m := NewDedupMailer(stub, store, time.Hour)
	ctx := WithEventKey(context.Background(), "e1")
	payload := newTestPayload("a@x.io")

	// Another worker claimed the same mail of the same event and hasn't committed it yet.
	other := NewDedupMailer(&mailerStub{onSend: func() {
		err := m.Send(ctx, payload)
		require.ErrorIs(t, err, ErrMailInFlight)
	}}, store, time.Hour)

	require.NoError(t, other.Send(ctx, payload))
	require.Empty(t, stub.calls)

	// Once committed, the same mail is skipped without an error.
	require.NoError(t, m.Send(ctx, payload))
	require.Empty(t, stub.calls)
}

// Deduplication is best effort, the mail still goes out if the store rejects the claim.
func TestDedupMailer_SendsWithoutDedupeIfTTLIsInvalid(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), 0)
	ctx := WithEventKey(context.Background(), "e1")

	require.NoError(t, m.Send(ctx, newTestPayload("a@x.io")))
	require.NoError(t, m.Send(ctx, newTestPayload("a@x.io")))

	require.Len(t, stub.calls, 2)
}

func TestDedupMailer_MailsDifferingInRepoRefOrContentTypeAreAllSent(t *testing.T) {
	t.Parallel()

	stub := &mailerStub{}
	m := NewDedupMailer(stub, idempotency.NewInMemory(), time.Hour)
	ctx := WithEventKey(context.Background(), "e1")

	base := newTestPayload("a@x.io")
	withRepo, withType := base, base
	withRepo.RepoRef = "space/repo"
	withType.ContentType = "text/plain"

	require.NoError(t, m.Send(ctx, base))
	require.NoError(t, m.Send(ctx, withRepo))
	require.NoError(t, m.Send(ctx, withType))
	require.Len(t, stub.calls, 3)
}
