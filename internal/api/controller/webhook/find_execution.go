// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// FindExecution finds a webhook execution.
func (c *Controller) FindExecution(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookID int64,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookID)
	if err != nil {
		return nil, err
	}
	// get the webhook execution and ensure it belongs to us
	webhookExecution, err := c.getWebhookExecutionVerifyOwnership(ctx, webhook.ID, webhookExecutionID)
	if err != nil {
		return nil, err
	}

	return webhookExecution, nil
}

func (c *Controller) getWebhookExecutionVerifyOwnership(ctx context.Context, webhookID int64,
	webhookExecutionID int64) (*types.WebhookExecution, error) {
	if webhookExecutionID <= 0 {
		return nil, usererror.BadRequest("A valid webhook execution ID must be provided.")
	}

	webhookExecution, err := c.webhookExecutionStore.Find(ctx, webhookExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find webhook execution with id %d: %w", webhookExecutionID, err)
	}

	// ensure the webhook execution actually belongs to the webhook
	if webhookID != webhookExecution.WebhookID {
		return nil, fmt.Errorf("webhook execution doesn't belong to requested webhook. Returning error %w",
			usererror.ErrNotFound)
	}

	return webhookExecution, nil
}
