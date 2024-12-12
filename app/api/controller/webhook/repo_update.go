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

// UpdateRepo updates an existing webhook.
func (c *Controller) UpdateRepo(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookIdentifier string,
	in *types.WebhookUpdateInput,
	signatureData *types.WebhookSignatureMetadata,
) (*types.Webhook, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	typ, err := c.preprocessor.PreprocessUpdateInput(session.Principal.Type, in, signatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess update input: %w", err)
	}

	return c.webhookService.Update(ctx, repo.ID, enum.WebhookParentRepo, webhookIdentifier, typ, in)
}
