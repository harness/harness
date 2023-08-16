// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type UpdateInput struct {
	DisplayName *string               `json:"display_name"`
	Description *string               `json:"description"`
	URL         *string               `json:"url"`
	Secret      *string               `json:"secret"`
	Enabled     *bool                 `json:"enabled"`
	Insecure    *bool                 `json:"insecure"`
	Triggers    []enum.WebhookTrigger `json:"triggers"`
	Internal    *bool                 `json:"internal"`
}

// Update updates an existing webhook.
func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	webhookID int64,
	in *UpdateInput,
) (*types.Webhook, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	// get the hook and ensure it belongs to us
	hook, err := c.getWebhookVerifyOwnership(ctx, repo.ID, webhookID)
	if err != nil {
		return nil, err
	}

	// validate input
	if err = checkUpdateInput(in, c.allowLoopback, c.allowPrivateNetwork || *in.Internal); err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	// update webhook struct (only for values that are provided)
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
		hook.Secret = *in.Secret
	}
	if in.Enabled != nil {
		hook.Enabled = *in.Enabled
	}
	if in.Insecure != nil {
		hook.Insecure = *in.Insecure
	}
	if in.Triggers != nil {
		hook.Triggers = deduplicateTriggers(in.Triggers)
	}

	if err = c.webhookStore.Update(ctx, hook); err != nil {
		return nil, err
	}

	return hook, nil
}

func checkUpdateInput(in *UpdateInput, allowLoopback bool, allowPrivateNetwork bool) error {
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
		if err := checkURL(*in.URL, allowLoopback, allowPrivateNetwork); err != nil {
			return err
		}
	}
	if in.Secret != nil {
		if err := checkSecret(*in.Secret); err != nil {
			return err
		}
	}
	if in.Triggers != nil {
		if err := checkTriggers(in.Triggers); err != nil {
			return err
		}
	}

	return nil
}
