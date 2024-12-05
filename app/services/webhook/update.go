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
	"errors"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) sanitizeUpdateInput(in *types.WebhookUpdateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := check.Identifier(*in.Identifier); err != nil {
			return err
		}
	}
	if in.DisplayName != nil {
		if err := check.DisplayName(*in.DisplayName); err != nil {
			return err
		}
	}
	if in.Description != nil {
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}
	if in.URL != nil {
		// internal is set to false as internal webhooks cannot be updated
		if err := CheckURL(*in.URL, s.config.AllowLoopback, s.config.AllowPrivateNetwork, false); err != nil {
			return err
		}
	}
	if in.Secret != nil {
		if err := CheckSecret(*in.Secret); err != nil {
			return err
		}
	}
	if in.Triggers != nil {
		if err := CheckTriggers(in.Triggers); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Update(
	ctx context.Context,
	parentID int64,
	parentType enum.WebhookParent,
	webhookIdentifier string,
	typ enum.WebhookType,
	in *types.WebhookUpdateInput,
) (*types.Webhook, error) {
	hook, err := s.GetWebhookVerifyOwnership(ctx, parentID, parentType, webhookIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook ownership: %w", err)
	}

	if err := s.sanitizeUpdateInput(in); err != nil {
		return nil, err
	}

	if typ != hook.Type {
		return nil, errors.New("changing type is not allowed")
	}

	// update webhook struct (only for values that are provided)
	if in.Identifier != nil {
		hook.Identifier = *in.Identifier
	}
	if in.DisplayName != nil {
		hook.DisplayName = *in.DisplayName
	}
	if in.Description != nil {
		hook.Description = *in.Description
	}
	if in.URL != nil {
		hook.URL = *in.URL
	}
	if in.Secret != nil {
		encryptedSecret, err := s.encrypter.Encrypt(*in.Secret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		hook.Secret = string(encryptedSecret)
	}
	if in.Enabled != nil {
		hook.Enabled = *in.Enabled
	}
	if in.Insecure != nil {
		hook.Insecure = *in.Insecure
	}
	if in.Triggers != nil {
		hook.Triggers = DeduplicateTriggers(in.Triggers)
	}

	if err := s.webhookStore.Update(ctx, hook); err != nil {
		return nil, err
	}

	return hook, nil
}
