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

package infraprovider

import (
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type Factory interface {
	GetInfraProvider(providerType enum.InfraProviderType) (InfraProvider, error)
}

type factory struct {
	providers map[enum.InfraProviderType]InfraProvider
}

func NewFactory(dockerProvider *DockerProvider) Factory {
	providers := make(map[enum.InfraProviderType]InfraProvider)
	providers[enum.InfraProviderTypeDocker] = dockerProvider
	return &factory{providers: providers}
}

func (f *factory) GetInfraProvider(providerType enum.InfraProviderType) (InfraProvider, error) {
	val := f.providers[providerType]
	if val == nil {
		return nil, fmt.Errorf("unknown infra provider type: %s", providerType)
	}
	return val, nil
}
