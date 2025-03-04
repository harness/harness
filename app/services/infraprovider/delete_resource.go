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
	"net/http"

	"github.com/harness/gitness/app/api/usererror"
)

func (c *Service) DeleteResource(
	ctx context.Context,
	spaceID int64,
	infraProviderConfigIdentifier string,
	identifier string,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		infraProviderConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, spaceID,
			infraProviderConfigIdentifier)
		if err != nil {
			return fmt.Errorf("failed to find infra config %s for deleting resource: %w",
				infraProviderConfigIdentifier, err)
		}

		infraProviderResource, err := c.infraProviderResourceStore.FindByConfigAndIdentifier(ctx, spaceID,
			infraProviderConfig.ID, identifier)
		if err != nil {
			return fmt.Errorf("failed to find infra resource %s with config %s for deleting resource: %w",
				identifier, infraProviderConfigIdentifier, err)
		}

		activeGitspaces, err := c.gitspaceConfigStore.ListActiveConfigsForInfraProviderResource(ctx,
			infraProviderResource.ID)
		if err != nil {
			return fmt.Errorf("failed to list active configs for infra resource %s for deleting resource: %w",
				identifier, err)
		}

		if len(activeGitspaces) > 0 {
			return usererror.NewWithPayload(http.StatusForbidden, fmt.Sprintf("There are %d active gitspace "+
				"configs for infra resource %s, expected 0", len(activeGitspaces), identifier))
		}

		return c.infraProviderResourceStore.Delete(ctx, infraProviderResource.ID)
	})

	if err != nil {
		return fmt.Errorf("failed to complete txn for deleting the infra resource %s: %w", identifier, err)
	}

	return nil
}
