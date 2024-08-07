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

package migrate

import (
	"context"
	"fmt"
	"time"

	webhookpkg "github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// Webhook is webhook migrate.
type Webhook struct {
	// webhook configs
	allowLoopback       bool
	allowPrivateNetwork bool

	tx           dbtx.Transactor
	webhookStore store.WebhookStore
}

func NewWebhook(
	config webhook.Config,
	tx dbtx.Transactor,
	webhookStore store.WebhookStore,
) *Webhook {
	return &Webhook{
		allowLoopback:       config.AllowLoopback,
		allowPrivateNetwork: config.AllowPrivateNetwork,
		tx:                  tx,
		webhookStore:        webhookStore,
	}
}

func (migrate Webhook) Import(
	ctx context.Context,
	migrator types.Principal,
	repo *types.Repository,
	extWebhooks []*ExternalWebhook,
) ([]*types.Webhook, error) {
	now := time.Now().UnixMilli()
	hooks := make([]*types.Webhook, len(extWebhooks))

	// sanitize and convert webhooks
	for i, whook := range extWebhooks {
		triggers := webhookpkg.ConvertTriggers(whook.Events)
		err := sanitizeWebhook(whook, triggers, migrate.allowLoopback, migrate.allowPrivateNetwork)
		if err != nil {
			return nil, fmt.Errorf("failed to sanitize external webhook input: %w", err)
		}

		// create new webhook object
		hook := &types.Webhook{
			ID:         0, // the ID will be populated in the data layer
			Version:    0, // the Version will be populated in the data layer
			CreatedBy:  migrator.ID,
			Created:    now,
			Updated:    now,
			ParentID:   repo.ID,
			ParentType: enum.WebhookParentRepo,

			// user input
			Identifier:            whook.Identifier,
			DisplayName:           whook.Identifier,
			URL:                   whook.Target,
			Enabled:               whook.Active,
			Insecure:              whook.SkipVerify,
			Triggers:              webhookpkg.DeduplicateTriggers(triggers),
			LatestExecutionResult: nil,
		}

		hooks[i] = hook
	}

	err := migrate.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, hook := range hooks {
			err := migrate.webhookStore.Create(ctx, hook)
			if err != nil {
				return fmt.Errorf("failed to store webhook: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store external webhooks: %w", err)
	}

	return hooks, nil
}

func sanitizeWebhook(
	in *ExternalWebhook,
	triggers []enum.WebhookTrigger,
	allowLoopback bool,
	allowPrivateNetwork bool,
) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	if err := webhookpkg.CheckURL(in.Target, allowLoopback, allowPrivateNetwork); err != nil {
		return err
	}

	if err := webhookpkg.CheckTriggers(triggers); err != nil { //nolint:revive
		return err
	}

	return nil
}
