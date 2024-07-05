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

import (
	"github.com/harness/gitness/infraprovider/enum"

	"github.com/guregu/null"
)

type InfraProviderConfig struct {
	ID         int64                    `json:"-"`
	Identifier string                   `json:"identifier"`
	Name       string                   `json:"name"`
	Type       enum.InfraProviderType   `json:"type"`
	Metadata   map[string]string        `json:"metadata"`
	Resources  []*InfraProviderResource `json:"resources"`
	SpaceID    int64                    `json:"-"`
	SpacePath  string                   `json:"space_path"`
	Created    int64                    `json:"created"`
	Updated    int64                    `json:"updated"`
}

type InfraProviderResource struct {
	ID                            int64                  `json:"-"`
	Identifier                    string                 `json:"identifier"`
	Name                          string                 `json:"name"`
	InfraProviderConfigID         int64                  `json:"-"`
	InfraProviderConfigIdentifier string                 `json:"config_identifier"`
	CPU                           null.String            `json:"cpu"`
	Memory                        null.String            `json:"memory"`
	Disk                          null.String            `json:"disk"`
	Network                       null.String            `json:"network"`
	Region                        string                 `json:"region"`
	Metadata                      map[string]string      `json:"metadata"`
	GatewayHost                   null.String            `json:"gateway_host"`
	GatewayPort                   null.String            `json:"gateway_port"`
	TemplateID                    null.Int               `json:"-"`
	TemplateIdentifier            null.String            `json:"template_identifier"`
	SpaceID                       int64                  `json:"-"`
	SpacePath                     string                 `json:"space_path"`
	InfraProviderType             enum.InfraProviderType `json:"infra_provider_type"`
	Created                       int64                  `json:"created"`
	Updated                       int64                  `json:"updated"`
}

type InfraProviderTemplate struct {
	ID                            int64  `json:"-"`
	Identifier                    string `json:"identifier"`
	InfraProviderConfigID         string `json:"-"`
	InfraProviderConfigIdentifier string `json:"config_identifier"`
	Description                   string `json:"description"`
	Data                          string `json:"data"`
	Version                       int64  `json:"-"`
	SpaceID                       int64  `json:"space_id"`
	SpacePath                     string `json:"space_path"`
	Created                       int64  `json:"created"`
	Updated                       int64  `json:"updated"`
}
