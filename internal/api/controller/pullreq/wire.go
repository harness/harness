// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB, authorizer authz.Authorizer,
	pullReqStore store.PullReqStore, pullReqActivityStore store.PullReqActivityStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore,
	rpcClient gitrpc.Interface) *Controller {
	return NewController(db, authorizer,
		pullReqStore, pullReqActivityStore, repoStore, principalStore,
		rpcClient)
}
