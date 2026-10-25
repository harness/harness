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

package events

import (
	"context"
	"fmt"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const (
	// DecisionMadeEvent is emitted after the AI gateway answered a decision batch (not from cache).
	DecisionMadeEvent events.EventType = "decision_made"
	// QuotaExhaustedEvent is emitted when a request was refused because the free model quota is used up.
	QuotaExhaustedEvent events.EventType = "quota_exhausted"
	// PaidModelDetectedEvent is emitted when a response reported cost for a model that must be free.
	PaidModelDetectedEvent events.EventType = "paid_model_detected"
)

type (
	DecisionMadePayload struct {
		SessionID string `json:"session_id"`
		Model     string `json:"model"`
		Fallback  bool   `json:"fallback"`
		Cached    bool   `json:"cached"`
	}

	QuotaExhaustedPayload struct {
		Class   string `json:"class"`
		Daily   bool   `json:"daily"`
		RetryAt int64  `json:"retry_at"` // unix milliseconds
	}

	PaidModelDetectedPayload struct {
		Model   string  `json:"model"`
		CostUSD float64 `json:"cost_usd"`
	}
)

func (r *Reporter) EmitDecisionMade(ctx context.Context, payload *DecisionMadePayload) error {
	if payload == nil {
		return fmt.Errorf("payload can not be nil for event type %s", DecisionMadeEvent)
	}
	return r.emit(ctx, DecisionMadeEvent, payload)
}

func (r *Reporter) EmitQuotaExhausted(ctx context.Context, payload *QuotaExhaustedPayload) error {
	if payload == nil {
		return fmt.Errorf("payload can not be nil for event type %s", QuotaExhaustedEvent)
	}
	return r.emit(ctx, QuotaExhaustedEvent, payload)
}

func (r *Reporter) EmitPaidModelDetected(ctx context.Context, payload *PaidModelDetectedPayload) error {
	if payload == nil {
		return fmt.Errorf("payload can not be nil for event type %s", PaidModelDetectedEvent)
	}
	return r.emit(ctx, PaidModelDetectedEvent, payload)
}

func (r *Reporter) emit(ctx context.Context, event events.EventType, payload any) error {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, event, payload)
	if err != nil {
		return fmt.Errorf("failed to send %s event: %w", event, err)
	}

	log.Ctx(ctx).Debug().Msgf("reported %v event with id '%s'", event, eventID)

	return nil
}

func (r *Reader) RegisterDecisionMade(
	fn events.HandlerFunc[*DecisionMadePayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, DecisionMadeEvent, fn, opts...)
}

func (r *Reader) RegisterQuotaExhausted(
	fn events.HandlerFunc[*QuotaExhaustedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, QuotaExhaustedEvent, fn, opts...)
}

func (r *Reader) RegisterPaidModelDetected(
	fn events.HandlerFunc[*PaidModelDetectedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, PaidModelDetectedEvent, fn, opts...)
}
