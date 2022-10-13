// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
)

type Controller struct {
	authorizer authz.Authorizer
	spaceStore store.SpaceStore
	repoStore  store.RepoStore
	saStore    store.ServiceAccountStore
}

func NewController(authorizer authz.Authorizer, spaceStore store.SpaceStore,
	repoStore store.RepoStore, saStore store.ServiceAccountStore) *Controller {
	return &Controller{
		authorizer: authorizer,
		spaceStore: spaceStore,
		repoStore:  repoStore,
		saStore:    saStore,
	}
}
