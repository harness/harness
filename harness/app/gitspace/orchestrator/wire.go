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

package orchestrator

import (
	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/platformconnector"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/secret"
	"github.com/harness/gitness/app/services/gitspacesettings"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideOrchestrator,
)

func ProvideOrchestrator(
	scm *scm.SCM,
	platformConnector platformconnector.PlatformConnector,
	infraProvisioner infrastructure.InfraProvisioner,
	containerOrchestratorFactor container.Factory,
	reporter *events.Reporter,
	config *Config,
	ideFactory ide.Factory,
	secretResolverFactory *secret.ResolverFactory,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	gitspaceConfigStore store.GitspaceConfigStore,
	settingsService gitspacesettings.Service,
	spaceStore store.SpaceStore,
	infraProviderSvc *infraprovider.Service,
) Orchestrator {
	return NewOrchestrator(
		scm,
		platformConnector,
		infraProvisioner,
		containerOrchestratorFactor,
		reporter,
		config,
		ideFactory,
		secretResolverFactory,
		gitspaceInstanceStore,
		gitspaceConfigStore,
		settingsService,
		spaceStore,
		infraProviderSvc,
	)
}
