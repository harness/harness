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

const GatewayHealthHealthy = "healthy"
const GatewayHealthUnhealthy = "unhealthy"

type CDEGatewayStats struct {
	Name        string `json:"name"`
	GroupName   string `json:"group_name"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	Health      string `json:"health"`
	EnvoyHealth string `json:"envoy_health"`
	Version     string `json:"version"`
}

type CDEGateway struct {
	CDEGatewayStats
	SpaceID                       int64  `json:"space_id,omitempty"`
	SpacePath                     string `json:"space_path"`
	InfraProviderConfigID         int64  `json:"infra_provider_config_id,omitempty"`
	InfraProviderConfigIdentifier string `json:"infra_provider_config_identifier"`
	OverallHealth                 string `json:"overall_health,omitempty"`
	Created                       int64  `json:"created"`
	Updated                       int64  `json:"updated"`
}

type CDEGatewayFilter struct {
	Health                     string  `json:"health,omitempty"`
	HealthReportValidityInMins int     `json:"health_report_validity_in_mins,omitempty"`
	InfraProviderConfigIDs     []int64 `json:"infra_provider_config_ids,omitempty"`
}
