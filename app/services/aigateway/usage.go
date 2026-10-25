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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// WithStores makes the quota durable and shared between replicas and records usage.
// Without stores the quota is kept in memory and usage isn't recorded.
func (g *Gateway) WithStores(quotaStore store.AIQuotaStore, usageStore store.AIUsageStore) *Gateway {
	if quotaStore != nil {
		g.planner = NewStorePlanner(g.config.DailyRequestLimit, g.config.perReplicaRPM(), nil, quotaStore)
	}
	g.usage = usageStore
	return g
}

// recordUsage stores one usage record for the requests made to a model, failed ones included.
// Errors are logged and never fail the call. Nothing is recorded when no request was sent.
func (g *Gateway) recordUsage(
	ctx context.Context,
	class Class,
	model string,
	requests int,
	tokensIn, tokensOut int64,
	cost float64,
) {
	if g.usage == nil || requests <= 0 {
		return
	}
	// the request may have failed because ctx was cancelled; the record is still owed
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), quotaStoreTimeout)
	defer cancel()

	spaceID, repoID := AttributionFrom(ctx)
	err := g.usage.Create(ctx, &types.AIUsage{
		SpaceID:   spaceID,
		RepoID:    repoID,
		Feature:   string(class),
		Model:     model,
		Requests:  requests,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   cost,
		Created:   time.Now().UnixMilli(),
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("model", model).Msg("ai gateway: failed to record usage")
	}
}
