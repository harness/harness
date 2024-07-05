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
	"github.com/harness/gitness/infraprovider"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideEmbeddedDockerOrchestrator,
	ProvideVSCodeWebService,
	ProvideVSCodeService,
)

func ProvideEmbeddedDockerOrchestrator(
	dockerClientFactory *infraprovider.DockerClientFactory,
	vsCodeService *VSCode,
	vsCodeWebService *VSCodeWeb,
	config *Config,
) Orchestrator {
	return NewEmbeddedDockerOrchestrator(
		dockerClientFactory,
		vsCodeService,
		vsCodeWebService,
		config,
	)
}

func ProvideVSCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return NewVsCodeWebService(config)
}

func ProvideVSCodeService() *VSCode {
	return NewVsCodeService()
}
