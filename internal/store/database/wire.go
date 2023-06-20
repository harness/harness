// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/migrate"
	"github.com/harness/gitness/store/database"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideDatabase,
	ProvidePrincipalStore,
	ProvidePrincipalInfoView,
	ProvidePathStore,
	ProvideSpaceStore,
	ProvideRepoStore,
	ProvideRepoGitInfoView,
	ProvideTokenStore,
	ProvidePullReqStore,
	ProvidePullReqActivityStore,
	ProvideCodeCommentView,
	ProvidePullReqReviewStore,
	ProvidePullReqReviewerStore,
	ProvideWebhookStore,
	ProvideWebhookExecutionStore,
	ProvideCheckStore,
	ProvideReqCheckStore,
)

// helper function to setup the database by performing automated
// database migration steps.
func migrator(ctx context.Context, db *sqlx.DB) error {
	return migrate.Migrate(ctx, db)
}

// ProvideDatabase provides a database connection.
func ProvideDatabase(ctx context.Context, config database.Config) (*sqlx.DB, error) {
	return database.ConnectAndMigrate(
		ctx,
		config.Driver,
		config.Datasource,
		migrator,
	)
}

// ProvidePrincipalStore provides a principal store.
func ProvidePrincipalStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) store.PrincipalStore {
	return NewPrincipalStore(db, uidTransformation)
}

// ProvidePrincipalInfoView provides a principal info store.
func ProvidePrincipalInfoView(db *sqlx.DB) store.PrincipalInfoView {
	return NewPrincipalInfoView(db)
}

// ProvidePathStore provides a path store.
func ProvidePathStore(db *sqlx.DB, pathTransformation store.PathTransformation) store.PathStore {
	return NewPathStore(db, pathTransformation)
}

// ProvideSpaceStore provides a space store.
func ProvideSpaceStore(db *sqlx.DB, pathCache store.PathCache) store.SpaceStore {
	return NewSpaceStore(db, pathCache)
}

// ProvideRepoStore provides a repo store.
func ProvideRepoStore(db *sqlx.DB, pathCache store.PathCache) store.RepoStore {
	return NewRepoStore(db, pathCache)
}

// ProvideRepoGitInfoView provides a repo git UID view.
func ProvideRepoGitInfoView(db *sqlx.DB) store.RepoGitInfoView {
	return NewRepoGitInfoView(db)
}

// ProvideTokenStore provides a token store.
func ProvideTokenStore(db *sqlx.DB) store.TokenStore {
	return NewTokenStore(db)
}

// ProvidePullReqStore provides a pull request store.
func ProvidePullReqStore(db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache) store.PullReqStore {
	return NewPullReqStore(db, principalInfoCache)
}

// ProvidePullReqActivityStore provides a pull request activity store.
func ProvidePullReqActivityStore(db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache) store.PullReqActivityStore {
	return NewPullReqActivityStore(db, principalInfoCache)
}

// ProvideCodeCommentView provides a code comment view.
func ProvideCodeCommentView(db *sqlx.DB) store.CodeCommentView {
	return NewCodeCommentView(db)
}

// ProvidePullReqReviewStore provides a pull request review store.
func ProvidePullReqReviewStore(db *sqlx.DB) store.PullReqReviewStore {
	return NewPullReqReviewStore(db)
}

// ProvidePullReqReviewerStore provides a pull request reviewer store.
func ProvidePullReqReviewerStore(db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache) store.PullReqReviewerStore {
	return NewPullReqReviewerStore(db, principalInfoCache)
}

// ProvideWebhookStore provides a webhook store.
func ProvideWebhookStore(db *sqlx.DB) store.WebhookStore {
	return NewWebhookStore(db)
}

// ProvideWebhookExecutionStore provides a webhook execution store.
func ProvideWebhookExecutionStore(db *sqlx.DB) store.WebhookExecutionStore {
	return NewWebhookExecutionStore(db)
}

// ProvideCheckStore provides a status check result store.
func ProvideCheckStore(db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.CheckStore {
	return NewCheckStore(db, principalInfoCache)
}

// ProvideReqCheckStore provides a required status check store.
func ProvideReqCheckStore(db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.ReqCheckStore {
	return NewReqCheckStore(db, principalInfoCache)
}
