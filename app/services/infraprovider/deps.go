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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// The interfaces below capture exactly what this service needs from the
// infraprovider factory/providers. Depending on these narrow, lightweight
// interfaces (instead of the concrete docker-heavy infraprovider package) keeps
// docker/docker out of the import graph of every consumer of *Service (e.g.
// code-api, which never provisions infra). The concrete infraprovider.Factory
// and infraprovider.InfraProvider satisfy these structurally (a thin adapter is
// provided at wire time so GetInfraProvider returns the narrow type).

// Provider is the subset of infraprovider.InfraProvider used by this service.
type Provider interface {
	UpdateParams(
		inputParameters []types.InfraProviderParameter,
		configMetaData map[string]any,
	) ([]types.InfraProviderParameter, error)
	ValidateParams(inputParameters []types.InfraProviderParameter) error
	TemplateParams() []types.InfraProviderParameterSchema
	UpdateConfig(infraProviderConfig *types.InfraProviderConfig) (*types.InfraProviderConfig, error)
	ValidateConfig(infraProviderConfig *types.InfraProviderConfig) error
	GenerateSetupYAML(infraProviderConfig *types.InfraProviderConfig) (string, error)
}

// ProviderFactory resolves a Provider by type. It mirrors the concrete
// infraprovider.Factory but returns the narrow Provider so this package does not
// import the heavy infraprovider package.
type ProviderFactory interface {
	GetInfraProvider(providerType enum.InfraProviderType) (Provider, error)
}
