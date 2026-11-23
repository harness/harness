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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harness/gitness/idempotency"

	"github.com/rs/zerolog/log"
)

// DefaultDedupTTL is how long a sent mail is remembered.
const DefaultDedupTTL = 24 * time.Hour

// inFlightTTL is how long a claim lives while the mail is being sent.
const inFlightTTL = 30 * time.Second

// ErrMailInFlight is returned when another worker is sending the same mail of the same event.
var ErrMailInFlight = errors.New("mail of this event is already being sent")

type eventKeyCtxKey struct{}

// WithEventKey returns a context carrying the key of the event being processed.
func WithEventKey(ctx context.Context, eventKey string) context.Context {
	return context.WithValue(ctx, eventKeyCtxKey{}, eventKey)
}

func eventKeyFrom(ctx context.Context) string {
	key, _ := ctx.Value(eventKeyCtxKey{}).(string)
	return key
}

// DedupMailer makes sending idempotent per (event, mail).
type DedupMailer struct {
	inner Mailer
	store idempotency.Store
	ttl   time.Duration
}

func NewDedupMailer(inner Mailer, store idempotency.Store, ttl time.Duration) DedupMailer {
	return DedupMailer{inner: inner, store: store, ttl: ttl}
}

func (m DedupMailer) Send(ctx context.Context, mailPayload Payload) error {
	eventKey := eventKeyFrom(ctx)
	if eventKey == "" {
		return m.inner.Send(ctx, mailPayload)
	}

	key := dedupKey(eventKey, mailPayload)

	state, err := m.store.Claim(ctx, key, min(inFlightTTL, m.ttl))
	if err != nil {
		// Favor delivery over deduplication.
		log.Ctx(ctx).Warn().Err(err).Msg("failed to claim mail in sent store, sending without deduplication")
		return m.inner.Send(ctx, mailPayload)
	}

	switch state {
	case idempotency.StateDone:
		log.Ctx(ctx).Info().Str("event_key", eventKey).Msg("mail already sent for event, skipping")
		return nil
	case idempotency.StateInFlight:
		// The other send may still fail, so retry the event instead of acknowledging it.
		return fmt.Errorf("%w: event %s", ErrMailInFlight, eventKey)
	case idempotency.StateClaimed:
	}

	err = m.inner.Send(ctx, mailPayload)
	if err != nil {
		// ctx may be canceled (e.g. shutdown), the release still has to go through.
		if errRelease := m.store.Release(context.WithoutCancel(ctx), key); errRelease != nil {
			log.Ctx(ctx).Warn().Err(errRelease).Msg("failed to release mail claim after send failure")
		}
		return err
	}

	if errCommit := m.store.Commit(context.WithoutCancel(ctx), key, m.ttl); errCommit != nil {
		log.Ctx(ctx).Warn().Err(errCommit).Msg("failed to commit mail claim after send, a redelivery may resend it")
	}

	return nil
}

func dedupKey(eventKey string, p Payload) string {
	to := append([]string(nil), p.ToRecipients...)
	cc := append([]string(nil), p.CCRecipients...)
	sort.Strings(to)
	sort.Strings(cc)

	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s",
		strings.Join(to, ","), strings.Join(cc, ","), p.Subject, p.Body, p.ContentType, p.RepoRef)

	return "notification:sent:" + eventKey + ":" + hex.EncodeToString(h.Sum(nil))
}
