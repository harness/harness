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

package serviceaccount

import (
	"context"

	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/store"
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
