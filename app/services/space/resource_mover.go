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

package space

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"

	"github.com/rs/zerolog/log"
)

// ResourceMover defines the interface for moving space resources.
// Different implementations exist for different services based on their schema.
type ResourceMover interface {
	// MoveResources moves all resources from source space to target space within a transaction.
	// Returns counts of moved resources.
	MoveResources(ctx context.Context, sourceSpaceID, targetSpaceID int64) (MoveResourcesOutput, error)
}

// CodeResourceMover handles moving only core gitness tables (repos, labels, rules, webhooks).
// Used by services that don't have registry tables (gitness-server, cde-manager).
type CodeResourceMover struct {
	repoStore    store.RepoStore
	labelStore   store.LabelStore
	rulesStore   store.RuleStore
	webhookStore store.WebhookStore
}

// NewCodeResourceMover creates a mover that only handles core gitness tables.
func NewCodeResourceMover(
	repoStore store.RepoStore,
	labelStore store.LabelStore,
	rulesStore store.RuleStore,
	webhookStore store.WebhookStore,
) *CodeResourceMover {
	return &CodeResourceMover{
		repoStore:    repoStore,
		labelStore:   labelStore,
		rulesStore:   rulesStore,
		webhookStore: webhookStore,
	}
}

// MoveResources moves core gitness resources.
func (m *CodeResourceMover) MoveResources(
	ctx context.Context,
	sourceSpaceID int64,
	targetSpaceID int64,
) (MoveResourcesOutput, error) {
	var output MoveResourcesOutput
	var err error

	output.RepoCount, err = m.repoStore.UpdateParent(ctx, sourceSpaceID, targetSpaceID)
	if err != nil {
		return output, fmt.Errorf("failed to move repos: %w", err)
	}

	output.LabelCount, err = m.labelStore.UpdateParentSpace(ctx, sourceSpaceID, targetSpaceID)
	if err != nil {
		return output, fmt.Errorf("failed to update labels: %w", err)
	}

	output.RuleCount, err = m.rulesStore.UpdateParentSpace(ctx, sourceSpaceID, targetSpaceID)
	if err != nil {
		return output, fmt.Errorf("failed to update rules: %w", err)
	}

	output.WebhookCount, err = m.webhookStore.UpdateParentSpace(ctx, sourceSpaceID, targetSpaceID)
	if err != nil {
		return output, fmt.Errorf("failed to update webhooks: %w", err)
	}

	// Log what core resources were moved
	log.Ctx(ctx).Info().Msgf(
		"Moved core resources: repos=%d, labels=%d, rules=%d, webhooks=%d",
		output.RepoCount, output.LabelCount, output.RuleCount, output.WebhookCount,
	)

	return output, nil
}
