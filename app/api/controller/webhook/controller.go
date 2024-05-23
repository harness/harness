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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	allowLoopback       bool
	allowPrivateNetwork bool

	authorizer            authz.Authorizer
	webhookStore          store.WebhookStore
	webhookExecutionStore store.WebhookExecutionStore
	repoStore             store.RepoStore
	webhookService        *webhook.Service
	encrypter             encrypt.Encrypter
}

func NewController(
	allowLoopback bool,
	allowPrivateNetwork bool,
	authorizer authz.Authorizer,
	webhookStore store.WebhookStore,
	webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore,
	webhookService *webhook.Service,
	encrypter encrypt.Encrypter,
) *Controller {
	return &Controller{
		allowLoopback:         allowLoopback,
		allowPrivateNetwork:   allowPrivateNetwork,
		authorizer:            authorizer,
		webhookStore:          webhookStore,
		webhookExecutionStore: webhookExecutionStore,
		repoStore:             repoStore,
		webhookService:        webhookService,
		encrypter:             encrypter,
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

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
	}

	return repo, nil
}
