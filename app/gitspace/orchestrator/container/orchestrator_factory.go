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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type Factory interface {
	GetContainerOrchestrator(providerType enum.InfraProviderType) (Orchestrator, error)
}
type factory struct {
	containerOrchestrators map[enum.InfraProviderType]Orchestrator
}

func NewFactory(embeddedDockerOrchestrator EmbeddedDockerOrchestrator) Factory {
	containerOrchestrators := make(map[enum.InfraProviderType]Orchestrator)
	containerOrchestrators[enum.InfraProviderTypeDocker] = &embeddedDockerOrchestrator
	return &factory{containerOrchestrators: containerOrchestrators}
}

func (f *factory) GetContainerOrchestrator(infraProviderType enum.InfraProviderType) (Orchestrator, error) {
	val, exist := f.containerOrchestrators[infraProviderType]
	if !exist {
		return nil, fmt.Errorf("unsupported infra provider type: %s", infraProviderType)
	}

	return val, nil
}
