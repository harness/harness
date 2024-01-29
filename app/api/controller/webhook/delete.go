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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes an existing webhook.
func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
	allowDeletingInternal bool,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return err
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookIdentifier)
	if err != nil {
		return err
	}
	if webhook.Internal && !allowDeletingInternal {
		return ErrInternalWebhookOperationNotAllowed
	}
	// delete webhook
	return c.webhookStore.Delete(ctx, webhook.ID)
}
