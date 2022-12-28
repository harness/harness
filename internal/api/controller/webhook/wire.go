// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/webhook"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(config *types.Config, db *sqlx.DB, authorizer authz.Authorizer,
	webhookStore store.WebhookStore, webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore, webhookServer *webhook.Server) *Controller {
	return NewController(config.Webhook.AllowLoopback, config.Webhook.AllowPrivateNetwork,
		db, authorizer, webhookStore, webhookExecutionStore, repoStore, webhookServer)
}
