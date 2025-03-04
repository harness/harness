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
	"github.com/harness/gitness/types"
)

func (c *Service) DeleteConfig(ctx context.Context, space *types.SpaceCore, identifier string) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		infraProviderConfig, err := c.Find(ctx, space, identifier)
		if err != nil {
			return fmt.Errorf("could not find infra provider config %s to delete: %w", identifier, err)
		}
		if len(infraProviderConfig.Resources) > 0 {
			return usererror.Newf(http.StatusForbidden, "There are %d resources in this config. Deletion "+
				"not allowed until all resources are deleted.", len(infraProviderConfig.Resources))
		}
		return c.infraProviderConfigStore.Delete(ctx, infraProviderConfig.ID)
	})
	if err != nil {
		return err
	}
	return nil
}
