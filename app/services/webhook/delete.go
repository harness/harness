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

	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Delete deletes an existing webhook.
func (s *Service) Delete(
	ctx context.Context,
	principal *types.Principal,
	webhookIdentifier string,
	parentResource ParentResource,
	allowDeletingInternal bool,
) error {
	hook, err := s.GetWebhookVerifyOwnership(ctx, parentResource.ID, parentResource.Type, webhookIdentifier)
	if err != nil {
		return err
	}

	if hook.Type == enum.WebhookTypeInternal && !allowDeletingInternal {
		return ErrInternalWebhookOperationNotAllowed
	}

	if err := s.webhookStore.Delete(ctx, hook.ID); err != nil {
		return err
	}

	resourceType, nameKey := getWebhookAuditInfo(parentResource.Type)
	err = s.auditService.Log(ctx,
		*principal,
		audit.NewResource(resourceType, hook.Identifier, nameKey, parentResource.Identifier),
		audit.ActionDeleted,
		parentResource.Path,
		audit.WithOldObject(hook),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete webhook operation: %s", err)
	}

	s.sendSSE(ctx, parentResource, enum.SSETypeWebhookDeleted, hook)

	return nil
}
