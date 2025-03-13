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

	registrystore "github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types"
)

type RegistryWebhookExecutorStore struct {
	webhookStore          registrystore.WebhooksRepository
	webhookExecutionStore registrystore.WebhooksExecutionRepository
}

func (s *RegistryWebhookExecutorStore) Find(ctx context.Context, id int64) (*types.WebhookExecutionCore, error) {
	return s.webhookExecutionStore.Find(ctx, id)
}

func (s *RegistryWebhookExecutorStore) ListWebhooks(
	ctx context.Context,
	parents []types.WebhookParentInfo,
) ([]*types.WebhookCore, error) {
	return s.webhookStore.ListAllByRegistry(ctx, parents)
}

func (s *RegistryWebhookExecutorStore) ListForTrigger(
	ctx context.Context,
	triggerID string,
) ([]*types.WebhookExecutionCore, error) {
	return s.webhookExecutionStore.ListForTrigger(ctx, triggerID)
}

func (s *RegistryWebhookExecutorStore) CreateWebhookExecution(
	ctx context.Context,
	hook *types.WebhookExecutionCore,
) error {
	return s.webhookExecutionStore.Create(ctx, hook)
}

func (s *RegistryWebhookExecutorStore) UpdateOptLock(
	ctx context.Context, hook *types.WebhookCore,
	execution *types.WebhookExecutionCore,
) (*types.WebhookCore, error) {
	fn := func(hook *types.WebhookCore) error {
		hook.LatestExecutionResult = &execution.Result
		return nil
	}
	return s.webhookStore.UpdateOptLock(ctx, hook, fn)
}

func (s *RegistryWebhookExecutorStore) FindWebhook(
	ctx context.Context,
	id int64,
) (*types.WebhookCore, error) {
	return s.webhookStore.Find(ctx, id)
}
