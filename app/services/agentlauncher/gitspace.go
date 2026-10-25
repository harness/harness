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

package agentlauncher

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

// AgentGitspaceIdentifier is the identifier of the gitspace config that runs the fix agents of the
// repositories of a space. An administrator creates it once per space (the launcher never creates gitspaces:
// they need an infra provider resource and credentials that only a human can choose).
const AgentGitspaceIdentifier = "harness-agent"

// StoreGitspaceResolver looks up the agent gitspace config by its well known identifier in the space of the
// repository.
type StoreGitspaceResolver struct {
	Configs store.GitspaceConfigStore
}

// Resolve implements GitspaceResolver.
func (r StoreGitspaceResolver) Resolve(ctx context.Context, repo *types.RepositoryCore) (*types.GitspaceConfig, error) {
	cfg, err := r.Configs.FindByIdentifier(ctx, repo.ParentID, AgentGitspaceIdentifier)
	if errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return nil, fmt.Errorf("%w: space %d has no gitspace config %q", ErrNoGitspace, repo.ParentID, AgentGitspaceIdentifier)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find agent gitspace config: %w", err)
	}
	if cfg.IsDeleted || cfg.IsMarkedForDeletion {
		return nil, fmt.Errorf("%w: gitspace config %q is deleted", ErrNoGitspace, AgentGitspaceIdentifier)
	}
	return cfg, nil
}
