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

package types

import "github.com/harness/gitness/types/enum"

type InfraProviderParameterSchema struct {
	Name         string
	Description  string
	DefaultValue string
	Required     bool
	Secret       bool
	Editable     bool
}

type InfraProviderParameter struct {
	Name  string
	Value string
}

type PortMapping struct {
	// PublishedPort is the port on which the container will be listening.
	PublishedPort int
	// ForwardedPort is the port on the infra to which the PublishedPort is forwarded.
	ForwardedPort int
}

type Infrastructure struct {
	// Identifier identifies the provisioned infra.
	Identifier string
	// SpaceID for the resource key.
	SpaceID int64
	// SpacePath for the resource key.
	SpacePath string
	// GitspaceConfigIdentifier is the gitspace config for which the infra is provisioned.
	GitspaceConfigIdentifier string
	// GitspaceInstanceIdentifier is the gitspace instance for which the infra is provisioned.
	GitspaceInstanceIdentifier string
	// ProviderType specifies the type of the infra provider.
	ProviderType enum.InfraProviderType
	// InputParameters which are required by the provider to provision the infra.
	InputParameters []InfraProviderParameter
	// Status of the infra.
	Status enum.InfraStatus
	// Host through which the infra can be accessed.
	Host      string
	ProxyHost string
	// AgentPort on which the agent can be accessed to orchestrate containers.
	AgentPort int
	ProxyPort int
	// Storage is the name of the volume or disk created for the resource.
	Storage string
	// GitspacePortMappings contains the ports assigned for every requested port.
	GitspacePortMappings map[int]*PortMapping
}
