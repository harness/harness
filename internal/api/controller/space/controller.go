// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
)

type Controller struct {
	gitBaseURL     string
	spaceCheck     check.Space
	authorizer     authz.Authorizer
	spaceStore     store.SpaceStore
	repoStore      store.RepoStore
	principalStore store.PrincipalStore
}

func NewController(gitBaseURL string, spaceCheck check.Space, authorizer authz.Authorizer, spaceStore store.SpaceStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore) *Controller {
	return &Controller{
		gitBaseURL:     gitBaseURL,
		spaceCheck:     spaceCheck,
		authorizer:     authorizer,
		spaceStore:     spaceStore,
		repoStore:      repoStore,
		principalStore: principalStore,
	}
}
