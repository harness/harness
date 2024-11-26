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

const (
	UnknownPlatformConnectorType        PlatformConnectorType = "unknown"
	ArtifactoryPlatformConnectorType    PlatformConnectorType = "Artifactory"
	DockerRegistryPlatformConnectorType PlatformConnectorType = "DockerRegistry"

	UnknownPlatformConnectorAuthType          PlatformConnectorAuthType = "unknown"
	UserNamePasswordPlatformConnectorAuthType PlatformConnectorAuthType = "UsernamePassword"
	AnonymousPlatformConnectorAuthType        PlatformConnectorAuthType = "Anonymous"
)

var (
	platformConnectorTypeMapping = map[string]PlatformConnectorType{
		ArtifactoryPlatformConnectorType.String():    ArtifactoryPlatformConnectorType,
		DockerRegistryPlatformConnectorType.String(): DockerRegistryPlatformConnectorType,
	}

	platformConnectorAuthTypeMapping = map[string]PlatformConnectorAuthType{
		UserNamePasswordPlatformConnectorAuthType.String(): UserNamePasswordPlatformConnectorAuthType,
		AnonymousPlatformConnectorAuthType.String():        AnonymousPlatformConnectorAuthType,
	}
)

type PlatformConnectorType string

func (t PlatformConnectorType) String() string { return string(t) }

func ToPlatformConnectorType(s string) PlatformConnectorType {
	if val, ok := platformConnectorTypeMapping[s]; ok {
		return val
	}

	return UnknownPlatformConnectorType
}

type PlatformConnectorAuthType string

func (t PlatformConnectorAuthType) String() string { return string(t) }

func ToPlatformConnectorAuthType(s string) PlatformConnectorAuthType {
	if val, ok := platformConnectorAuthTypeMapping[s]; ok {
		return val
	}

	return UnknownPlatformConnectorAuthType
}

type PlatformConnector struct {
	ID            string
	Name          string
	ConnectorSpec PlatformConnectorSpec
}

type PlatformConnectorSpec struct {
	Type PlatformConnectorType
	// ArtifactoryURL is for ArtifactoryPlatformConnectorType
	ArtifactoryURL string
	// DockerRegistryURL is for DockerRegistryPlatformConnectorType
	DockerRegistryURL string
	AuthSpec          PlatformConnectorAuthSpec
	EnabledProxy      bool
}

// PlatformConnectorAuthSpec provide auth details.
// PlatformConnectorAuthSpec is empty for AnonymousPlatformConnectorAuthType.
type PlatformConnectorAuthSpec struct {
	AuthType PlatformConnectorAuthType
	// userName can be empty when userName is encrypted.
	UserName *MaskSecret
	// UserNameRef can be empty when userName is not encrypted
	UserNameRef string
	Password    *MaskSecret
	PasswordRef string
}

func (c PlatformConnectorSpec) ExtractRegistryURL() string {
	switch c.Type {
	case DockerRegistryPlatformConnectorType:
		return c.DockerRegistryURL
	case ArtifactoryPlatformConnectorType:
		return c.ArtifactoryURL
	case UnknownPlatformConnectorType:
		return ""
	default:
		return ""
	}
}

func (c PlatformConnectorAuthSpec) ExtractUserName() string {
	if c.AuthType == UserNamePasswordPlatformConnectorAuthType {
		return c.UserName.Value()
	}

	return ""
}

func (c PlatformConnectorAuthSpec) ExtractUserNameRef() string {
	if c.AuthType == UserNamePasswordPlatformConnectorAuthType {
		return c.UserNameRef
	}

	return ""
}

func (c PlatformConnectorAuthSpec) ExtractPasswordRef() string {
	if c.AuthType == UserNamePasswordPlatformConnectorAuthType {
		return c.PasswordRef
	}

	return ""
}

func (c PlatformConnectorAuthSpec) ExtractPassword() string {
	if c.AuthType == UserNamePasswordPlatformConnectorAuthType {
		return c.Password.Value()
	}

	return ""
}
