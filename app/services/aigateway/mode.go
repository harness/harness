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
	"fmt"

	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// repoSettingsGetter is the subset of settings.Service used by the ModeResolver.
type repoSettingsGetter interface {
	RepoGet(ctx context.Context, repoID int64, key settings.Key, out any) (bool, error)
}

// ModeResolver resolves the effective AI mode of a repository.
type ModeResolver struct {
	settings     repoSettingsGetter
	publicAccess publicaccess.Service
}

func NewModeResolver(settingsSvc *settings.Service, publicAccess publicaccess.Service) *ModeResolver {
	return &ModeResolver{settings: settingsSvc, publicAccess: publicAccess}
}

// Resolve returns the repo's explicit AI mode if one is set, otherwise the default for its visibility
// (public: full, private: decisions-only). An invalid stored value is an error, never silently widened.
func (r *ModeResolver) Resolve(ctx context.Context, repo *types.RepositoryCore) (enum.AIMode, error) {
	var stored enum.AIMode
	found, err := r.settings.RepoGet(ctx, repo.ID, settings.KeyAIMode, &stored)
	if err != nil {
		return "", fmt.Errorf("failed to get ai mode setting: %w", err)
	}

	if found && stored != "" {
		mode, ok := stored.Sanitize()
		if !ok {
			return "", fmt.Errorf("invalid ai mode %q stored for repo %d", stored, repo.ID)
		}
		return mode, nil
	}

	isPublic, err := r.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to check repo visibility: %w", err)
	}

	return settings.DefaultAIModeForVisibility(isPublic), nil
}

// AllowsDecisions reports whether compact metadata may be sent to the decision model.
func AllowsDecisions(mode enum.AIMode) bool {
	return mode == enum.AIModeFull || mode == enum.AIModeDecisionsOnly
}

// AllowsGeneration reports whether (redacted) content may be sent to generative models.
func AllowsGeneration(mode enum.AIMode) bool {
	return mode == enum.AIModeFull
}

// AllowsAgents reports whether fix agents may run on the repo.
func AllowsAgents(mode enum.AIMode) bool {
	return mode == enum.AIModeFull
}
