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

package webhook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateSpace updates an existing webhook.
func (c *Controller) UpdateSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	webhookIdentifier string,
	in *types.WebhookUpdateInput,
) (*types.Webhook, error) {
	space, err := c.getSpaceCheckAccess(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	allowModifyingInternal, err := c.preprocessor.PreprocessUpdateInput(session.Principal.Type, in)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess update input: %w", err)
	}

	return c.webhookService.Update(
		ctx, space.ID, enum.WebhookParentSpace, webhookIdentifier, allowModifyingInternal, in,
	)
}
