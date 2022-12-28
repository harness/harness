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
)

// List returns the webhooks from the provided repository.
func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.WebhookFilter,
) ([]*types.Webhook, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, err
	}

	count, err := c.webhookStore.Count(ctx, enum.WebhookParentRepo, repo.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count webhooks for repo with id %d: %w", repo.ID, err)
	}

	webhooks, err := c.webhookStore.List(ctx, enum.WebhookParentRepo, repo.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list webhooks for repo with id %d: %w", repo.ID, err)
	}

	return webhooks, count, nil
}
