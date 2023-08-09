// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB, urlProvider *url.Provider, uidCheck check.PathUID, authorizer authz.Authorizer,
	pathStore store.PathStore, pipelineStore store.PipelineStore, secretStore store.SecretStore,
	spaceStore store.SpaceStore, repoStore store.RepoStore, principalStore store.PrincipalStore,
	repoCtrl *repo.Controller, membershipStore store.MembershipStore,
) *Controller {
	return NewController(db, urlProvider, uidCheck, authorizer,
		pathStore, pipelineStore, secretStore, spaceStore, repoStore,
		principalStore, repoCtrl, membershipStore)
}
