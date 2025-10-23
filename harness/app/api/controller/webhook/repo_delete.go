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
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/types/enum"
)

// DeleteRepo deletes an existing webhook.
func (c *Controller) DeleteRepo(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	return c.webhookService.Delete(
		ctx, &session.Principal, webhookIdentifier, webhook.ParentResource{
			ID:         repo.ID,
			Identifier: repo.Identifier,
			Type:       enum.WebhookParentRepo,
			Path:       paths.Parent(repo.Path),
		},
		c.preprocessor.IsInternalCall(session.Principal.Type),
	)
}
