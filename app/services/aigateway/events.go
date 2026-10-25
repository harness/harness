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

package aigateway

import (
	"context"
	"errors"

	aievents "github.com/harness/gitness/app/events/ai"

	"github.com/rs/zerolog/log"
)

// EventReporter publishes AI gateway events. Without a reporter the gateway emits no events.
type EventReporter interface {
	EmitDecisionMade(ctx context.Context, payload *aievents.DecisionMadePayload) error
	EmitQuotaExhausted(ctx context.Context, payload *aievents.QuotaExhaustedPayload) error
	EmitPaidModelDetected(ctx context.Context, payload *aievents.PaidModelDetectedPayload) error
}

// SetEventReporter sets the reporter for gateway events. Passing nil disables events.
func (g *Gateway) SetEventReporter(r EventReporter) {
	g.events = r
}

func (g *Gateway) emitDecisionMade(ctx context.Context, sessionID string, res *DecideResult) {
	if g.events == nil {
		return
	}
	logEventError(ctx, g.events.EmitDecisionMade(ctx, &aievents.DecisionMadePayload{
		SessionID: sessionID,
		Model:     res.Model,
		Fallback:  res.Fallback,
		Cached:    res.Cached,
	}))
}

func (g *Gateway) emitQuotaExhausted(ctx context.Context, err error) {
	var qe *QuotaError
	if g.events == nil || !errors.As(err, &qe) {
		return
	}
	logEventError(ctx, g.events.EmitQuotaExhausted(ctx, &aievents.QuotaExhaustedPayload{
		Class:   string(qe.Class),
		Daily:   qe.Daily,
		RetryAt: qe.RetryAt.UnixMilli(),
	}))
}

func (g *Gateway) emitPaidModelDetected(ctx context.Context, model string, cost float64) {
	if g.events == nil {
		return
	}
	logEventError(ctx, g.events.EmitPaidModelDetected(ctx, &aievents.PaidModelDetectedPayload{
		Model:   model,
		CostUSD: cost,
	}))
}

// logEventError keeps event failures out of the request path: a lost event must not fail the call.
func logEventError(ctx context.Context, err error) {
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to report event")
	}
}
