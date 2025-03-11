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
