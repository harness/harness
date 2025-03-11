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

	gitnessstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

type GitnessWebhookExecutorStore struct {
	webhookStore          gitnessstore.WebhookStore
	webhookExecutionStore gitnessstore.WebhookExecutionStore
}

func (s *GitnessWebhookExecutorStore) Find(ctx context.Context, id int64) (*types.WebhookExecutionCore, error) {
	execution, err := s.webhookExecutionStore.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	executionCore := GitnessWebhookExecutionToWebhookExecutionCore(execution)
	return executionCore, nil
}

func (s *GitnessWebhookExecutorStore) ListWebhooks(
	ctx context.Context,
	parents []types.WebhookParentInfo,
) ([]*types.WebhookCore, error) {
	webhooks, err := s.webhookStore.List(ctx, parents, &types.WebhookFilter{})
	if err != nil {
		return nil, err
	}
	webhooksCore := GitnessWebhooksToWebhooksCore(webhooks)
	return webhooksCore, nil
}

func (s *GitnessWebhookExecutorStore) ListForTrigger(
	ctx context.Context,
	triggerID string,
) ([]*types.WebhookExecutionCore, error) {
	executions, err := s.webhookExecutionStore.ListForTrigger(ctx, triggerID)
	if err != nil {
		return nil, err
	}
	webhookExecutionsCore := make([]*types.WebhookExecutionCore, 0)
	for _, e := range executions {
		executionCore := GitnessWebhookExecutionToWebhookExecutionCore(e)
		webhookExecutionsCore = append(webhookExecutionsCore, executionCore)
	}
	return webhookExecutionsCore, nil
}

func (s *GitnessWebhookExecutorStore) CreateWebhookExecution(
	ctx context.Context,
	hook *types.WebhookExecutionCore,
) error {
	webhookExecution := CoreWebhookExecutionToGitnessWebhookExecution(hook)
	return s.webhookExecutionStore.Create(ctx, webhookExecution)
}

func (s *GitnessWebhookExecutorStore) UpdateOptLock(
	ctx context.Context, hook *types.WebhookCore,
	execution *types.WebhookExecutionCore,
) (*types.WebhookCore, error) {
	webhook := CoreWebhookToGitnessWebhook(hook)
	fn := func(hook *types.Webhook) error {
		hook.LatestExecutionResult = &execution.Result
		return nil
	}
	gitnessWebhook, err := s.webhookStore.UpdateOptLock(ctx, webhook, fn)
	if err != nil {
		return nil, err
	}
	webhookCore := GitnessWebhookToWebhookCore(gitnessWebhook)
	return webhookCore, err
}

func (s *GitnessWebhookExecutorStore) FindWebhook(
	ctx context.Context,
	id int64,
) (*types.WebhookCore, error) {
	webhook, err := s.webhookStore.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	webhookCore := GitnessWebhookToWebhookCore(webhook)
	return webhookCore, nil
}
