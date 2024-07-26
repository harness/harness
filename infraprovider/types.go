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

import "github.com/harness/gitness/infraprovider/enum"

type ParameterSchema struct {
	Name         string
	Description  string
	DefaultValue string
	Required     bool
	Secret       bool
	Editable     bool
}

type Parameter struct {
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
	// ResourceKey is the key for which the infra is provisioned.
	ResourceKey string
	// SpaceID for the resource key.
	SpaceID int64
	// SpacePath for the resource key.
	SpacePath string
	// ProviderType specifies the type of the infra provider.
	ProviderType enum.InfraProviderType
	// Parameters which are required by the provider to provision the infra.
	Parameters []Parameter
	// Status of the infra.
	Status enum.InfraStatus
	// Host through which the infra can be accessed for all purposes.
	Host string
	// Port on which the infra can be accessed to orchestrate containers.
	Port int
	// Storage is the name of the volume or disk created for the resource.
	Storage string
	// PortMappings contains the ports assigned for every requested port.
	PortMappings map[int]*PortMapping
}
