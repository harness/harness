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

type InfraProvisioned struct {
	ID                      int64
	GitspaceInstanceID      int64
	InfraProviderType       enum.InfraProviderType
	InfraProviderResourceID int64
	SpaceID                 int64
	Created                 int64
	Updated                 int64
	ResponseMetadata        *string
	InputParams             string
	InfraStatus             enum.InfraStatus
	ServerHostIP            string
	ServerHostPort          string
	ProxyHost               string
	ProxyPort               int32
}

type InfraProvisionedGatewayView struct {
	GitspaceInstanceIdentifier string
	SpaceID                    int64
	ServerHostIP               string
	ServerHostPort             string
}

type InfraProvisionedUpdateGatewayRequest struct {
	GitspaceInstanceIdentifier string `json:"gitspace_id"`
	SpaceID                    int64  `json:"space_id"`
	GatewayHost                string `json:"gateway_host"`
	GatewayPort                int32  `json:"gateway_port"`
}

type InfraProvisionedResponse struct {
	ServerHost                 string `json:"server_host"`
	ServerPort                 string `json:"server_port"`
	GitspaceInstanceIdentifier string `json:"gitspace_id"`
	SpaceID                    int64  `json:"space_id"`
}
