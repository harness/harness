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
	"errors"

	"github.com/harness/gitness/app/store"
)

// NoopResourceMover is a no-op implementation of ResourceMover.
// Used in gitness standalone where space move is not supported.
type NoopResourceMover struct{}

// MoveResources returns an error indicating the operation is not supported.
func (m *NoopResourceMover) MoveResources(
	ctx context.Context,
	sourceSpaceID int64,
	targetSpaceID int64,
) (MoveResourcesOutput, error) {
	return MoveResourcesOutput{}, errors.New("space move not supported in this deployment")
}

// ProvideNoopResourceMover returns a no-op ResourceMover for gitness standalone.
// Move feature is not supported in gitness standalone.
func ProvideNoopResourceMover() ResourceMover {
	return &NoopResourceMover{}
}

// ProvideCodeResourceMover provides a resource mover that handles only core gitness tables.
// Used by cde-manager.
func ProvideCodeResourceMover(
	repoStore store.RepoStore,
	labelStore store.LabelStore,
	rulesStore store.RuleStore,
	webhookStore store.WebhookStore,
) ResourceMover {
	return NewCodeResourceMover(repoStore, labelStore, rulesStore, webhookStore)
}
