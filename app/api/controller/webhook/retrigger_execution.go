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

	"github.com/rs/zerolog/log"
)

// RetriggerExecution retriggers an existing webhook execution.
func (c *Controller) RetriggerExecution(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	// get the webhook and ensure it belongs to us
	webhook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookIdentifier)
	if err != nil {
		return nil, err
	}

	// get the webhookexecution and ensure it belongs to us
	webhookExecution, err := c.getWebhookExecutionVerifyOwnership(ctx, webhook.ID, webhookExecutionID)
	if err != nil {
		return nil, err
	}

	// retrigger the execution ...
	executionResult, err := c.webhookService.RetriggerWebhookExecution(ctx, webhookExecution.ID)
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
