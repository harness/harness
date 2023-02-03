// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"

	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
)

type Controller struct {
	principalUIDCheck check.PrincipalUID
	authorizer        authz.Authorizer
	principalStore    store.PrincipalStore
	spaceStore        store.SpaceStore
	repoStore         store.RepoStore
	tokenStore        store.TokenStore
}

func NewController(principalUIDCheck check.PrincipalUID, authorizer authz.Authorizer,
	principalStore store.PrincipalStore, spaceStore store.SpaceStore, repoStore store.RepoStore,
	tokenStore store.TokenStore) *Controller {
	return &Controller{
		principalUIDCheck: principalUIDCheck,
		authorizer:        authorizer,
		principalStore:    principalStore,
		spaceStore:        spaceStore,
		repoStore:         repoStore,
		tokenStore:        tokenStore,
	}
}

func findServiceAccountFromUID(ctx context.Context,
	principalStore store.PrincipalStore, saUID string) (*types.ServiceAccount, error) {
	return principalStore.FindServiceAccountByUID(ctx, saUID)
}
