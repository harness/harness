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
	"time"

	"github.com/harness/gitness/types"
)

func (c *Service) ReportStats(
	ctx context.Context,
	spaceCore *types.SpaceCore,
	infraProviderConfig *types.InfraProviderConfig,
	in *types.CDEGatewayStats,
) error {
	gateway := types.CDEGateway{
		InfraProviderConfigID:         infraProviderConfig.ID,
		InfraProviderConfigIdentifier: infraProviderConfig.Identifier,
		SpaceID:                       spaceCore.ID,
		SpacePath:                     spaceCore.Path,
	}
	gateway.Name = in.Name
	gateway.GroupName = in.GroupName
	gateway.Region = in.Region
	gateway.Zone = in.Zone
	gateway.Version = in.Version
	gateway.Health = in.Health
	gateway.EnvoyHealth = in.EnvoyHealth
	gateway.Created = time.Now().UnixMilli()
	gateway.Updated = gateway.Created

	return c.gatewayStore.Upsert(ctx, &gateway)
}
