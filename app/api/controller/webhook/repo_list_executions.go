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
)

// ListExecutionsRepo returns the executions of the webhook.
func (c *Controller) ListExecutionsRepo(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
	filter *types.WebhookExecutionFilter,
) ([]*types.WebhookExecution, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	return c.webhookService.ListExecutions(
		ctx, repo.ID, enum.WebhookParentRepo, webhookIdentifier, filter)
}