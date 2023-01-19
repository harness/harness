// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	allowLoopback       bool
	allowPrivateNetwork bool

	db                    *sqlx.DB
	authorizer            authz.Authorizer
	webhookStore          store.WebhookStore
	webhookExecutionStore store.WebhookExecutionStore
	repoStore             store.RepoStore
	webhookService        *webhook.Service
}

func NewController(
	allowLoopback bool,
	allowPrivateNetwork bool,
	db *sqlx.DB,
	authorizer authz.Authorizer,
	webhookStore store.WebhookStore,
	webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore,
	webhookService *webhook.Service,
) *Controller {
	return &Controller{
		allowLoopback:       allowLoopback,
		allowPrivateNetwork: allowPrivateNetwork,

		db:                    db,
		authorizer:            authorizer,
		webhookStore:          webhookStore,
		webhookExecutionStore: webhookExecutionStore,
		repoStore:             repoStore,
		webhookService:        webhookService,
	}
}

func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session, repoRef string, reqPermission enum.Permission) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission, false); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
	}

	return repo, nil
}
