// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"github.com/google/wire"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(config *types.Config, repoCheck check.Repo, authorizer authz.Authorizer,
	spaceStore store.SpaceStore, repoStore store.RepoStore, saStore store.ServiceAccountStore,
	rpcClient gitrpc.Interface) *Controller {
	return NewController(config.Git.DefaultBranch, repoCheck, authorizer, spaceStore, repoStore, saStore, rpcClient)
}
