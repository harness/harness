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

	"github.com/harness/gitness/types"
)

func (c *Service) UpdateTemplate(ctx context.Context, template types.InfraProviderTemplate) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err := c.spaceFinder.FindByRef(ctx, template.SpacePath)
		if err != nil {
			return err
		}
		templateInDB, err := c.infraProviderTemplateStore.FindByIdentifier(ctx, space.ID, template.Identifier)
		if err != nil {
			return err
		}
		template.ID = templateInDB.ID
		template.SpaceID = space.ID
		if err = c.infraProviderTemplateStore.Update(ctx, &template); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete update txn for the infraprovider template %w", err)
	}
	return nil
}
