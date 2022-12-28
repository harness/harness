// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"time"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CreateInput struct {
	URL      string                `json:"url"`
	Secret   string                `json:"secret"`
	Enabled  bool                  `json:"enabled"`
	Insecure bool                  `json:"insecure"`
	Triggers []enum.WebhookTrigger `json:"triggers"`
}

// Create creates a new webhook.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.Webhook, error) {
	now := time.Now().UnixMilli()

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	// validate input
	err = checkCreateInput(in, c.allowLoopback, c.allowPrivateNetwork)
	if err != nil {
		return nil, err
	}

	// create new webhook object
	hook := &types.Webhook{
		ID:         0, // the ID will be populated in the data layer
		Version:    0, // the Version will be populated in the data layer
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		ParentID:   repo.ID,
		ParentType: enum.WebhookParentRepo,

		// user input
		URL:      in.URL,
		Secret:   in.Secret,
		Enabled:  in.Enabled,
		Insecure: in.Insecure,
		Triggers: deduplicateTriggers(in.Triggers),
	}

	err = c.webhookStore.Create(ctx, hook)
	if err != nil {
		return nil, err
	}

	return hook, nil
}

func checkCreateInput(in *CreateInput, allowLoopback bool, allowPrivateNetwork bool) error {
	if err := checkURL(in.URL, allowLoopback, allowPrivateNetwork); err != nil {
		return err
	}
	if err := checkSecret(in.Secret); err != nil {
		return err
	}
	if err := checkTriggers(in.Triggers); err != nil {
		return err
	}

	return nil
}
