// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// RetriggerExecution retriggers an existing webhook execution.
func (c *Controller) RetriggerExecution(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookID int64,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookID)
	if err != nil {
		return nil, err
	}

	// get the webhookexecution and ensure it belongs to us
	webhookExecution, err := c.getWebhookExecutionVerifyOwnership(ctx, webhook.ID, webhookExecutionID)
	if err != nil {
		return nil, err
	}

	// retrigger the execution ...
	executionResult, err := c.webhookServer.RetriggerWebhookExecution(ctx, webhookExecution.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrigger webhook execution: %w", err)
	}

	// log execution error so we have the necessary debug information if needed
	if executionResult.Err != nil {
		log.Ctx(ctx).Warn().Err(executionResult.Err).Msgf(
			"retrigger of webhhook %d execution %d (new id: %d) had an error",
			webhook.ID, webhookExecution.ID, executionResult.Execution.ID)
	}

	return executionResult.Execution, nil
}
