// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"time"

	"github.com/harness/gitness/internal/cache"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

const (
	postgres = "postgres"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideDatabase,
	ProvidePrincipalStore,
	ProvidePrincipalInfoView,
	ProvidePrincipalInfoCache,
	ProvideSpaceStore,
	ProvideRepoStore,
	ProvideTokenStore,
	ProvidePullReqStore,
	ProvidePullReqActivityStore,
	ProvidePullReqReviewStore,
	ProvidePullReqReviewerStore,
	ProvideWebhookStore,
	ProvideWebhookExecutionStore,
)

// ProvideDatabase provides a database connection.
func ProvideDatabase(ctx context.Context, config *types.Config) (*sqlx.DB, error) {
	return Connect(
		ctx,
		config.Database.Driver,
		config.Database.Datasource,
	)
}

// ProvidePrincipalStore provides a principal store.
func ProvidePrincipalStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) store.PrincipalStore {
	return NewPrincipalStore(db, uidTransformation)
}

// ProvidePrincipalInfoView provides a principal info store.
func ProvidePrincipalInfoView(db *sqlx.DB) store.PrincipalInfoView {
	return NewPrincipalInfoStore(db)
}

// ProvidePrincipalInfoCache provides a cache for storing types.PrincipalInfo objects.
func ProvidePrincipalInfoCache(getter store.PrincipalInfoView) *cache.Cache[int64, *types.PrincipalInfo] {
	return cache.New[int64, *types.PrincipalInfo](getter, 30*time.Second)
}

// ProvideSpaceStore provides a space store.
func ProvideSpaceStore(db *sqlx.DB, pathTransformation store.PathTransformation) store.SpaceStore {
	switch db.DriverName() {
	case postgres:
		return NewSpaceStore(db, pathTransformation)
	default:
		return NewSpaceStoreSync(
			NewSpaceStore(db, pathTransformation),
		)
	}
}

// ProvideRepoStore provides a repo store.
func ProvideRepoStore(db *sqlx.DB, pathTransformation store.PathTransformation) store.RepoStore {
	switch db.DriverName() {
	case postgres:
		return NewRepoStore(db, pathTransformation)
	default:
		return NewRepoStoreSync(
			NewRepoStore(db, pathTransformation),
		)
	}
}

// ProvideTokenStore provides a token store.
func ProvideTokenStore(db *sqlx.DB) store.TokenStore {
	return NewTokenStore(db)
}

// ProvidePullReqStore provides a pull request store.
func ProvidePullReqStore(db *sqlx.DB,
	principalInfoCache *cache.Cache[int64, *types.PrincipalInfo]) store.PullReqStore {
	return NewPullReqStore(db, principalInfoCache)
}

// ProvidePullReqActivityStore provides a pull request activity store.
func ProvidePullReqActivityStore(db *sqlx.DB,
	principalInfoCache *cache.Cache[int64, *types.PrincipalInfo]) store.PullReqActivityStore {
	return NewPullReqActivityStore(db, principalInfoCache)
}

// ProvidePullReqReviewStore provides a pull request review store.
func ProvidePullReqReviewStore(db *sqlx.DB) store.PullReqReviewStore {
	return NewPullReqReviewStore(db)
}

// ProvidePullReqReviewerStore provides a pull request reviewer store.
func ProvidePullReqReviewerStore(db *sqlx.DB,
	principalInfoCache *cache.Cache[int64, *types.PrincipalInfo]) store.PullReqReviewerStore {
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
