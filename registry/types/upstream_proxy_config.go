//  Copyright 2023 Harness, Inc.
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

import (
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

// UpstreamProxyConfig DTO object.
type UpstreamProxyConfig struct {
	ID                       int64
	RegistryID               int64
	Source                   string
	URL                      string
	AuthType                 string
	UserName                 string
	UserNameSecretIdentifier string
	UserNameSecretSpaceID    int64
	Password                 string
	SecretIdentifier         string
	SecretSpaceID            int64
	Token                    string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	CreatedBy                int64
	UpdatedBy                int64
}

type UpstreamProxy struct {
	ID                       int64
	RegistryUUID             string
	RegistryID               int64
	RepoKey                  string
	ParentID                 string
	PackageType              artifact.PackageType
	AllowedPattern           []string
	BlockedPattern           []string
	Config                   *RegistryConfig
	Source                   string
	RepoURL                  string
	RepoAuthType             string
	UserName                 string
	UserNameSecretIdentifier string
	UserNameSecretSpaceID    int64
	UserNameSecretSpacePath  string
	SecretIdentifier         string
	SecretSpaceID            int64
	SecretSpacePath          string
	Token                    string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	CreatedBy                int64
	UpdatedBy                int64
}
