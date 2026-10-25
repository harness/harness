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

package aigateway

import (
	"context"
	"errors"
	"time"

	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const storeTimeout = 10 * time.Second

// SetStore enables persistence of the catalog. The given ids are the decision models; every
// other model is stored with kind chat. Quarantines found in the store are restored into the
// guard, so a quarantine survives restarts. A nil store keeps the catalog in memory only.
func (c *Catalog) SetStore(ctx context.Context, s store.AIModelStore, decisionModels ...string) {
	if s == nil {
		return
	}

	c.mu.Lock()
	c.store = s
	c.decisionModels = make(map[string]struct{}, len(decisionModels))
	for _, id := range decisionModels {
		c.decisionModels[id] = struct{}{}
	}
	c.mu.Unlock()

	c.guard.setOnQuarantine(func(id, reason string) { c.persistQuarantine(id, reason) })

	stored, err := s.List(ctx, types.AIModelFilter{Status: enum.AIModelStatusQuarantined})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to load quarantined models")
		return
	}
	for _, m := range stored {
		c.guard.restoreQuarantine(m.ID, m.QuarantinedReason)
	}
}

// persist upserts the synced free models and retires stored models that are gone from the catalog.
// Failures are logged only: persistence must never break the in-memory catalog.
func (c *Catalog) persist(ctx context.Context, models map[string]Model) {
	c.mu.RLock()
	s := c.store
	c.mu.RUnlock()
	if s == nil {
		return
	}

	stored, err := s.List(ctx, types.AIModelFilter{})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to list stored models")
		return
	}
	existing := make(map[string]*types.AIModel, len(stored))
	for _, m := range stored {
		existing[m.ID] = m
	}

	now := time.Now().UnixMilli()
	for id, m := range models {
		row := &types.AIModel{
			ID:            id,
			Kind:          c.kindOf(id),
			ContextLength: m.ContextLength,
			SupportsTools: m.SupportsTools(),
			Status:        enum.AIModelStatusActive,
			LastSeen:      now,
		}
		if old, ok := existing[id]; ok {
			row.SuccessRateByTask = old.SuccessRateByTask
			row.P95LatencyMS = old.P95LatencyMS
			row.Created = old.Created
			if old.Status == enum.AIModelStatusDegraded {
				row.Status = old.Status
			}
		}
		if reason, ok := c.guard.QuarantineReason(id); ok {
			row.Status = enum.AIModelStatusQuarantined
			row.QuarantinedReason = reason
		}
		if err := s.Upsert(ctx, row); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("model", id).Msg("ai gateway: failed to persist model")
		}
	}

	for id, old := range existing {
		if _, ok := models[id]; ok {
			continue
		}
		if old.Status == enum.AIModelStatusRetired || old.Status == enum.AIModelStatusQuarantined {
			continue
		}
		if err := s.UpdateStatus(ctx, id, enum.AIModelStatusRetired, ""); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("model", id).Msg("ai gateway: failed to retire model")
		}
	}
}

func (c *Catalog) kindOf(id string) enum.AIModelKind {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.decisionModels[id]; ok {
		return enum.AIModelKindDecision
	}
	return enum.AIModelKindChat
}

// persistQuarantine stores a runtime quarantine. It has no caller context (the guard has none).
func (c *Catalog) persistQuarantine(id, reason string) {
	c.mu.RLock()
	s := c.store
	m, known := c.models[id]
	c.mu.RUnlock()
	if s == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	defer cancel()

	err := s.UpdateStatus(ctx, id, enum.AIModelStatusQuarantined, reason)
	if errors.Is(err, gitnessstore.ErrResourceNotFound) && known {
		err = s.Upsert(ctx, &types.AIModel{
			ID:                id,
			Kind:              c.kindOf(id),
			ContextLength:     m.ContextLength,
			SupportsTools:     m.SupportsTools(),
			Status:            enum.AIModelStatusQuarantined,
			QuarantinedReason: reason,
			LastSeen:          time.Now().UnixMilli(),
		})
	}
	if err != nil {
		log.Warn().Err(err).Str("model", id).Msg("ai gateway: failed to persist quarantine")
	}
}
