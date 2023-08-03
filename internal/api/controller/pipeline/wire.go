package pipeline

import (
	"github.com/google/wire"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB,
	uidCheck check.PathUID,
	pathStore store.PathStore,
	repoStore store.RepoStore,
	authorizer authz.Authorizer,
	pipelineStore store.PipelineStore,
	spaceStore store.SpaceStore,
) *Controller {
	return NewController(db, uidCheck, authorizer, pathStore, repoStore, pipelineStore, spaceStore)
}
