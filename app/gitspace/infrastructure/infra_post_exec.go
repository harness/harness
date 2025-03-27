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

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// PostInfraEventComplete performs action taken on completion of the submitted infra task which can be
// success or failure. It stores the infrastructure details in the db depending on the provisioning type.
func (i InfraProvisioner) PostInfraEventComplete(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	eventType enum.InfraEvent,
) error {
	infraProvider, err := i.getInfraProvider(infra.ProviderType)
	if err != nil {
		return err
	}

	if infraProvider.ProvisioningType() != enum.InfraProvisioningTypeNew {
		return nil
	}

	switch eventType {
	case enum.InfraEventProvision,
		enum.InfraEventDeprovision,
		enum.InfraEventStop,
		enum.InfraEventCleanup:
		return i.UpdateInfraProvisioned(ctx, gitspaceConfig.GitspaceInstance.ID, infra)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func (i InfraProvisioner) UpdateInfraProvisioned(
	ctx context.Context,
	gitspaceInstanceID int64,
	infrastructure types.Infrastructure,
) error {
	infraProvisionedLatest, err := i.infraProvisionedStore.FindLatestByGitspaceInstanceID(ctx, gitspaceInstanceID)
	if err != nil {
		return fmt.Errorf(
			"could not find latest infra provisioned entity for instance %d: %w", gitspaceInstanceID, err)
	}

	responseMetadata, err := json.Marshal(infrastructure)
	if err != nil {
		return fmt.Errorf("unable to marshal infra object %+v: %w", responseMetadata, err)
	}
	responseMetaDataJSON := string(responseMetadata)
	infraProvisionedLatest.InfraStatus = infrastructure.Status
	infraProvisionedLatest.ServerHostIP = infrastructure.AgentHost
	infraProvisionedLatest.ServerHostPort = strconv.Itoa(infrastructure.AgentPort)

	proxyHost := infrastructure.AgentHost
	if infrastructure.ProxyAgentHost != "" {
		proxyHost = infrastructure.ProxyAgentHost
	}
	infraProvisionedLatest.ProxyHost = proxyHost

	proxyPort := infrastructure.AgentPort
	if infrastructure.ProxyAgentPort != 0 {
		proxyPort = infrastructure.ProxyAgentPort
	}
	if proxyPort > math.MaxInt32 || proxyPort < math.MinInt32 {
		return fmt.Errorf("proxyPort value %d exceeds int32 range", proxyPort)
	}
	infraProvisionedLatest.ProxyPort = int32(proxyPort)

	infraProvisionedLatest.ResponseMetadata = &responseMetaDataJSON
	infraProvisionedLatest.Updated = time.Now().UnixMilli()

	infraProvisionedLatest.GatewayHost = infrastructure.GatewayHost
	err = i.infraProvisionedStore.Update(ctx, infraProvisionedLatest)
	if err != nil {
		return fmt.Errorf("unable to update infraProvisioned %d: %w", infraProvisionedLatest.ID, err)
	}
	return nil
}
