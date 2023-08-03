package execution

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db             *sqlx.DB
	authorizer     authz.Authorizer
	executionStore store.ExecutionStore
	repoStore      store.RepoStore
	spaceStore     store.SpaceStore
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
) *Controller {
	return &Controller{
		db:             db,
		authorizer:     authorizer,
		executionStore: executionStore,
		repoStore:      repoStore,
		spaceStore:     spaceStore,
	}
}
