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

// Package gitspacewire holds the wire providers that construct a
// *gitspace.Service from the concrete (docker/oras-heavy) orchestrator and ide
// implementations. It is kept separate from the gitspace package so that
// consumers of *gitspace.Service (e.g. code-api, which wires the service to
// nil) do not transitively import the heavy orchestrator/ide packages.
package gitspacewire

import (
	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	gitspacedeleteevents "github.com/harness/gitness/app/events/gitspacedelete"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/tokengenerator"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideGitspace,
)

func ProvideGitspace(
	tx dbtx.Transactor,
	gitspaceStore store.GitspaceConfigStore,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	eventReporter *gitspaceevents.Reporter,
	gitspaceEventStore store.GitspaceEventStore,
	spaceFinder refcache.SpaceFinder,
	infraProviderSvc *infraprovider.Service,
	orchestrator orchestrator.Orchestrator,
	scm *scm.SCM,
	config *types.Config,
	gitspaceDeleteEventReporter *gitspacedeleteevents.Reporter,
	ideFactory ide.Factory,
	spaceStore store.SpaceStore,
	tokenGenerator tokengenerator.TokenGenerator,
) *gitspace.Service {
	return gitspace.NewService(tx, gitspaceStore, gitspaceInstanceStore, eventReporter,
		gitspaceEventStore, spaceFinder, infraProviderSvc, orchestrator, scm, config,
		gitspaceDeleteEventReporter, ideFactoryAdapter{ideFactory}, spaceStore, tokenGenerator,
	)
}

// ideFactoryAdapter adapts the concrete ide.Factory (whose GetIDE returns the
// heavy ide.IDE) to the narrow gitspace.IDEFactory interface used by the service.
type ideFactoryAdapter struct {
	factory ide.Factory
}

func (a ideFactoryAdapter) GetIDE(ideType enum.IDEType) (gitspace.PluginURLGenerator, error) {
	return a.factory.GetIDE(ideType)
}
