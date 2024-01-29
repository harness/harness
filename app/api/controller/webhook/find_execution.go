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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// FindExecution finds a webhook execution.
func (c *Controller) FindExecution(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookIdentifier)
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
