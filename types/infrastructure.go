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

type InstanceInfo struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	IPAddress         string `json:"ip_address"`
	Port              int64  `json:"port"`
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	Provider          string `json:"provider"`
	PoolName          string `json:"pool_name"`
	Zone              string `json:"zone"`
	StorageIdentifier string `json:"storage_identifier"`
	CAKey             []byte `json:"ca_key"`
	CACert            []byte `json:"ca_cert"`
	TLSKey            []byte `json:"tls_key"`
	TLSCert           []byte `json:"tls_cert"`
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
	// ConfigMetadata contains the infra config metadata required by the infra provider to provision the infra.
	ConfigMetadata map[string]any
	// Status of the infra.
	Status enum.InfraStatus

	// AgentHost through which the infra can be accessed.
	AgentHost string
	// AgentPort on which the agent can be accessed to orchestrate containers.
	AgentPort int
	// ProxyAgentHost on which to connect to agent incase a proxy is used.
	ProxyAgentHost string
	ProxyAgentPort int
	// HostScheme is scheme to connect to the host e.g. https
	HostScheme string

	// GitspaceHost on which gitspace is accessible directly, without proxy being configured.
	GitspaceHost string
	// ProxyGitspaceHost on which gitspace is accessible through a proxy.
	ProxyGitspaceHost string
	GitspaceScheme    string

	// Storage is the name of the volume or disk created for the resource.
	Storage string
	// GitspacePortMappings contains the ports assigned for every requested port.
	GitspacePortMappings map[int]*PortMapping
	// VM Instance information from the init task
	// that are required for execute (start/stop, publish) and cleanup tasks
	// to make the runner stateless
	InstanceInfo InstanceInfo
	GatewayHost  string
}
