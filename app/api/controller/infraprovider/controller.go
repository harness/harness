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
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/types/enum"
)

const NoResourceIdentifier = ""

type ConfigInput struct {
	Identifier string                 `json:"identifier" yaml:"identifier"`
	SpaceRef   string                 `json:"space_ref" yaml:"space_ref"`
	Name       string                 `json:"name" yaml:"name"`
	Type       enum.InfraProviderType `json:"type" yaml:"type"`
	Metadata   map[string]any         `json:"metadata" yaml:"metadata"`
}

type ResourceInput struct {
	Identifier        string                 `json:"identifier" yaml:"identifier"`
	Name              string                 `json:"name" yaml:"name"`
	InfraProviderType enum.InfraProviderType `json:"infra_provider_type" yaml:"infra_provider_type"`
	CPU               *string                `json:"cpu" yaml:"cpu"`
	Memory            *string                `json:"memory" yaml:"memory"`
	Disk              *string                `json:"disk" yaml:"disk"`
	Network           *string                `json:"network" yaml:"network"`
	Region            string                 `json:"region" yaml:"region"`
	Metadata          map[string]string      `json:"metadata" yaml:"metadata"`
	GatewayHost       *string                `json:"gateway_host" yaml:"gateway_host"`
	GatewayPort       *string                `json:"gateway_port" yaml:"gateway_port"`
}

type AutoCreateInput struct {
	Config    ConfigInput     `json:"config" yaml:"config"`
	Resources []ResourceInput `json:"resources" yaml:"resources"`
}

type TemplateInput struct {
	Identifier  string `json:"identifier" yaml:"identifier"`
	Description string `json:"description" yaml:"description"`
	Data        string `json:"data" yaml:"data"`
}

type Controller struct {
	authorizer       authz.Authorizer
	spaceFinder      refcache.SpaceFinder
	infraproviderSvc *infraprovider.Service
}

func NewController(
	authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder,
	infraproviderSvc *infraprovider.Service,
) *Controller {
	return &Controller{
		authorizer:       authorizer,
		spaceFinder:      spaceFinder,
		infraproviderSvc: infraproviderSvc,
	}
}
