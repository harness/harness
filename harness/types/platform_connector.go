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
	ecrRegistryUserName = "AWS"

	UnknownPlatformConnectorType        PlatformConnectorType = "Unknown"
	ArtifactoryPlatformConnectorType    PlatformConnectorType = "Artifactory"
	DockerRegistryPlatformConnectorType PlatformConnectorType = "DockerRegistry"
	AWSPlatformConnectorType            PlatformConnectorType = "Aws"

	UnknownPlatformConnectorAuthType          PlatformConnectorAuthType = "unknown"
	UserNamePasswordPlatformConnectorAuthType PlatformConnectorAuthType = "UsernamePassword"
	AnonymousPlatformConnectorAuthType        PlatformConnectorAuthType = "Anonymous"

	UnknownAWSCredentialsType      AwsCredentialsType = "unknown"
	ManualConfigAWSCredentialsType AwsCredentialsType = "ManualConfig"
)

var (
	platformConnectorTypeMapping = map[string]PlatformConnectorType{
		ArtifactoryPlatformConnectorType.String():    ArtifactoryPlatformConnectorType,
		DockerRegistryPlatformConnectorType.String(): DockerRegistryPlatformConnectorType,
		AWSPlatformConnectorType.String():            AWSPlatformConnectorType,
	}

	platformConnectorAuthTypeMapping = map[string]PlatformConnectorAuthType{
		UserNamePasswordPlatformConnectorAuthType.String(): UserNamePasswordPlatformConnectorAuthType,
		AnonymousPlatformConnectorAuthType.String():        AnonymousPlatformConnectorAuthType,
	}

	awsCredentialsTypeMapping = map[string]AwsCredentialsType{
		ManualConfigAWSCredentialsType.String(): ManualConfigAWSCredentialsType,
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

type AwsCredentialsType string

func (t AwsCredentialsType) String() string { return string(t) }

func ToAwsCredentialsType(s string) AwsCredentialsType {
	if val, ok := awsCredentialsTypeMapping[s]; ok {
		return val
	}

	return UnknownAWSCredentialsType
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
	// AwsECRRegistryURL is for AWSPlatformConnectorType
	AwsECRRegistryURL string
	// AwsCredentials is for AWSPlatformConnectorType
	AwsCredentials AwsCredentials
	// AuthSpec is for ArtifactoryPlatformConnectorType and DockerRegistryPlatformConnectorType
	AuthSpec     PlatformConnectorAuthSpec
	EnabledProxy bool
}

type AwsCredentials struct {
	Type            AwsCredentialsType
	Region          string
	AccessToken     MaskSecret
	AccessKey       MaskSecret
	AccessKeyRef    string
	SecretKeyRef    string
	SecretKey       MaskSecret
	SessionTokenRef string
	SessionToken    MaskSecret
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
	case AWSPlatformConnectorType:
		return c.AwsECRRegistryURL
	case UnknownPlatformConnectorType:
		return ""
	default:
		return ""
	}
}

func (c PlatformConnectorSpec) ExtractRegistryUserName() string {
	if (c.Type == DockerRegistryPlatformConnectorType || c.Type == ArtifactoryPlatformConnectorType) &&
		c.AuthSpec.AuthType == UserNamePasswordPlatformConnectorAuthType {
		return c.AuthSpec.UserName.Value()
	} else if c.Type == AWSPlatformConnectorType {
		return ecrRegistryUserName
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

func (c PlatformConnectorSpec) ExtractRegistryAuth() string {
	if (c.Type == DockerRegistryPlatformConnectorType || c.Type == ArtifactoryPlatformConnectorType) &&
		c.AuthSpec.AuthType == UserNamePasswordPlatformConnectorAuthType &&
		c.AuthSpec.Password != nil {
		return c.AuthSpec.Password.Value()
	} else if c.Type == AWSPlatformConnectorType && c.AwsCredentials.AccessToken.Value() != "" {
		return c.AwsCredentials.AccessToken.Value()
	}
	return ""
}
