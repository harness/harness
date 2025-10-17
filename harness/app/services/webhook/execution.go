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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// FindExecution finds a webhook execution.
func (s *Service) FindExecution(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	webhook, err := s.GetWebhookVerifyOwnership(ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return nil, err
	}

	webhookExecution, err := s.GetWebhookExecutionVerifyOwnership(ctx, webhook.ID, webhookExecutionID)
	if err != nil {
		return nil, err
	}

	return webhookExecution, nil
}

// ListExecutions returns the executions of the webhook.
func (s *Service) ListExecutions(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
	filter *types.WebhookExecutionFilter,
) ([]*types.WebhookExecution, int64, error) {
	webhook, err := s.GetWebhookVerifyOwnership(ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to verify ownership for webhook %d: %w", webhook.ID, err)
	}

	total, err := s.webhookExecutionStore.CountForWebhook(ctx, webhook.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count webhook executions for webhook %d: %w", webhook.ID, err)
	}

	webhookExecutions, err := s.webhookExecutionStore.ListForWebhook(ctx, webhook.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list webhook executions for webhook %d: %w", webhook.ID, err)
	}

	return webhookExecutions, total, nil
}

// RetriggerExecution retriggers an existing webhook execution.
func (s *Service) RetriggerExecution(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
	webhookExecutionID int64,
) (*types.WebhookExecutionCore, error) {
	webhook, err := s.GetWebhookVerifyOwnership(
		ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return nil, err
	}

	webhookExecution, err := s.GetWebhookExecutionVerifyOwnership(
		ctx, webhook.ID, webhookExecutionID)
	if err != nil {
		return nil, err
	}

	executionResult, err := s.WebhookExecutor.RetriggerWebhookExecution(ctx, webhookExecution.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrigger webhook execution: %w", err)
	}

	if executionResult.Err != nil {
		log.Ctx(ctx).Warn().Err(executionResult.Err).Msgf(
			"retrigger of webhhook %d execution %d (new id: %d) had an error",
			webhook.ID, webhookExecution.ID, executionResult.Execution.ID)
	}

	return executionResult.Execution, nil
}
