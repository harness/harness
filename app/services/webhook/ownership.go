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
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) Find(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
) (*types.Webhook, error) {
	hook, err := s.GetWebhookVerifyOwnership(ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return nil, errors.NotFoundf("failed to find webhook %s: %q", webhookIdentifier, err)
	}

	return hook, nil
}

// GetWebhookVerifyOwnership gets the webhook and
// ensures it belongs to the scope with the specified id and type.
func (s *Service) GetWebhookVerifyOwnership(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
) (*types.Webhook, error) {
	// TODO: Remove once webhook identifier migration completed
	webhookID, err := strconv.ParseInt(webhookIdentifier, 10, 64)
	if (err == nil && webhookID <= 0) || len(strings.TrimSpace(webhookIdentifier)) == 0 {
		return nil, errors.InvalidArgument("A valid webhook identifier must be provided.")
	}

	var webhook *types.Webhook
	if err == nil {
		webhook, err = s.webhookStore.Find(ctx, webhookID)
	} else {
		webhook, err = s.webhookStore.FindByIdentifier(
			ctx, parentType, parentID, webhookIdentifier)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find webhook with identifier %q: %w", webhookIdentifier, err)
	}

	// ensure the webhook actually belongs to the repo
	if webhook.ParentType != parentType || webhook.ParentID != parentID {
		return nil, errors.NotFoundf("webhook doesn't belong to requested %s.", parentType)
	}

	return webhook, nil
}

// GetWebhookExecutionVerifyOwnership gets the webhook execution and
// ensures it belongs to the webhook with the specified id.
func (s *Service) GetWebhookExecutionVerifyOwnership(
	ctx context.Context,
	webhookID int64,
	webhookExecutionID int64,
) (*types.WebhookExecution, error) {
	if webhookExecutionID <= 0 {
		return nil, errors.InvalidArgument("A valid webhook execution ID must be provided.")
	}

	webhookExecution, err := s.webhookExecutionStore.Find(ctx, webhookExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find webhook execution with id %d: %w", webhookExecutionID, err)
	}

	// ensure the webhook execution actually belongs to the webhook
	if webhookID != webhookExecution.WebhookID {
		return nil, errors.NotFound("webhook execution doesn't belong to requested webhook")
	}

	return webhookExecution, nil
}
