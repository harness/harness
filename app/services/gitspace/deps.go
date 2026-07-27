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

package gitspace

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// The interfaces below capture exactly what this service needs from the
// gitspace orchestrator, ide factory and infraprovider service. Depending on
// these narrow, lightweight interfaces (instead of the concrete types) keeps
// the docker/oras-heavy orchestrator + ide packages out of the import graph of
// every consumer of *gitspace.Service (e.g. code-api, which wires this service
// to nil). The concrete implementations satisfy these interfaces structurally.

// InfraProviderFinder is the subset of *infraprovider.Service used here.
type InfraProviderFinder interface {
	Find(ctx context.Context, space *types.SpaceCore, identifier string) (*types.InfraProviderConfig, error)
}

// PluginURLGenerator is the subset of an IDE used here.
type PluginURLGenerator interface {
	GeneratePluginURL(projectName, gitspaceInstanceUID string) string
}

// IDEFactory resolves an IDE by type. It mirrors the concrete ide.Factory but
// returns the narrow PluginURLGenerator so this package does not import the
// heavy ide package. A thin adapter is provided at wire time.
type IDEFactory interface {
	GetIDE(ideType enum.IDEType) (PluginURLGenerator, error)
}

// Orchestrator is the subset of the gitspace orchestrator used by this service.
// The concrete orchestrator.Orchestrator satisfies it structurally.
type Orchestrator interface {
	TriggerStartGitspace(ctx context.Context, gitspaceConfig types.GitspaceConfig) *types.GitspaceError
	TriggerStopGitspace(ctx context.Context, gitspaceConfig types.GitspaceConfig) *types.GitspaceError
	TriggerDeleteGitspace(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		canDeleteUserData bool,
	) *types.GitspaceError
	TriggerCleanupInstanceResources(ctx context.Context, gitspaceConfig types.GitspaceConfig) error
}
