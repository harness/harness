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

package container

import (
	events "github.com/harness/gitness/app/events/gitspaceoperations"
	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideEmbeddedDockerOrchestrator,
	ProvideContainerOrchestratorFactory,
)

func ProvideEmbeddedDockerOrchestrator(
	dockerClientFactory *infraprovider.DockerClientFactory,
	statefulLogger *logutil.StatefulLogger,
	runArgProvider runarg.Provider,
	eventReporter *events.Reporter,
	config *types.Config,
) EmbeddedDockerOrchestrator {
	return NewEmbeddedDockerOrchestrator(
		dockerClientFactory,
		statefulLogger,
		runArgProvider,
		eventReporter,
		// devcontainer feature sources are user provided, so restrict which
		// addresses a feature download is allowed to reach.
		utils.NewFeatureDownloadTransport(netpolicy.Policy{
			AllowLoopback:       config.Gitspace.FeaturesAllowLoopback,
			AllowPrivateNetwork: config.Gitspace.FeaturesAllowPrivateNetwork,
			AllowLinkLocal:      config.Gitspace.FeaturesAllowLinkLocal,
		}),
	)
}

func ProvideContainerOrchestratorFactory(
	embeddedDockerOrchestrator EmbeddedDockerOrchestrator,
) Factory {
	return NewFactory(embeddedDockerOrchestrator)
}
