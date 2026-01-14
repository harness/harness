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
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/types"
)

func (c *Service) ListGateways(ctx context.Context, filter *types.CDEGatewayFilter) ([]*types.CDEGateway, error) {
	if filter == nil || len(filter.InfraProviderConfigIDs) == 0 {
		return nil, fmt.Errorf("cde-gateway filter is required")
	}

	if filter.HealthReportValidityInMins == 0 {
		filter.HealthReportValidityInMins = 5
	}

	gateways, err := c.gatewayStore.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateways: %w", err)
	}

	infraProviderConfigMap := make(map[int64]string)
	for _, gateway := range gateways {
		if _, ok := infraProviderConfigMap[gateway.InfraProviderConfigID]; !ok {
			infraProviderConfig, err := c.infraProviderConfigStore.Find(ctx, gateway.InfraProviderConfigID, false)
			if err != nil {
				return nil, fmt.Errorf("failed to find infra provider config %d while listing gateways: %w",
					gateway.InfraProviderConfigID, err)
			}
			infraProviderConfigMap[gateway.InfraProviderConfigID] = infraProviderConfig.Identifier
		}
		gateway.InfraProviderConfigIdentifier = infraProviderConfigMap[gateway.InfraProviderConfigID]

		spaceCore, err := c.spaceFinder.FindByID(ctx, gateway.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to find space %d while listing gateways: %w", gateway.SpaceID, err)
		}

		gateway.SpacePath = spaceCore.Path

		if gateway.Updated < time.Now().Add(-time.Duration(filter.HealthReportValidityInMins)*time.Minute).UnixMilli() {
			gateway.Health = types.GatewayHealthUnhealthy
			gateway.EnvoyHealth = types.GatewayHealthUnknown
		}

		if gateway.Health != types.GatewayHealthHealthy || gateway.EnvoyHealth != types.GatewayHealthHealthy {
			gateway.OverallHealth = types.GatewayHealthUnhealthy
		} else {
			gateway.OverallHealth = types.GatewayHealthHealthy
		}
	}

	return gateways, nil
}
