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

	"github.com/harness/gitness/types/enum"
)

// Delete deletes an existing webhook.
func (s *Service) Delete(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
	allowDeletingInternal bool,
) error {
	hook, err := s.GetWebhookVerifyOwnership(ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return err
	}

	if hook.Type == enum.WebhookTypeInternal && !allowDeletingInternal {
		return ErrInternalWebhookOperationNotAllowed
	}

	return s.webhookStore.Delete(ctx, hook.ID)
}
