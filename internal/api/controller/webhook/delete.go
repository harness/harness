// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes an existing webhook.
func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookID int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return err
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookID)
	if err != nil {
		return err
	}

	// delete webhook
	return c.webhookStore.Delete(ctx, webhook.ID)
}
