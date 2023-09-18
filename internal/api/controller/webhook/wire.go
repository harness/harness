// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(config webhook.Config, db *sqlx.DB, authorizer authz.Authorizer,
	webhookStore store.WebhookStore, webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore, webhookService *webhook.Service, encrypter encrypt.Encrypter) *Controller {
	return NewController(config.AllowLoopback, config.AllowPrivateNetwork,
		db, authorizer, webhookStore, webhookExecutionStore, repoStore, webhookService, encrypter)
}
