package execution

import (
	"github.com/google/wire"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
) *Controller {
	return NewController(db, authorizer, executionStore,
		repoStore,
		spaceStore)
}
