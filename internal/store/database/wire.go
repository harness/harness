// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

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
	ProvideUserStore,
	ProvideServiceAccountStore,
	ProvideServiceStore,
	ProvideSpaceStore,
	ProvideRepoStore,
	ProvideTokenStore,
	ProvidePullReqStore,
	ProvidePullReqActivityStore,
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

// ProvideUserStore provides a user store.
func ProvideUserStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) store.UserStore {
	return NewUserStore(db, uidTransformation)
}

// ProvideServiceAccountStore provides a service account store.
func ProvideServiceAccountStore(db *sqlx.DB,
	uidTransformation store.PrincipalUIDTransformation) store.ServiceAccountStore {
	return NewServiceAccountStore(db, uidTransformation)
}

// ProvideServiceStore provides a service store.
func ProvideServiceStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) store.ServiceStore {
	return NewServiceStore(db, uidTransformation)
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
func ProvidePullReqStore(db *sqlx.DB) store.PullReqStore {
	return NewPullReqStore(db)
}

// ProvidePullReqActivityStore provides a pull request activity store.
func ProvidePullReqActivityStore(db *sqlx.DB) store.PullReqActivityStore {
	return NewPullReqActivityStore(db)
}

// ProvideWebhookStore provides a webhook store.
func ProvideWebhookStore(db *sqlx.DB) store.WebhookStore {
	return NewWebhookStore(db)
}

// ProvideWebhookExecutionStore provides a webhook execution store.
func ProvideWebhookExecutionStore(db *sqlx.DB) store.WebhookExecutionStore {
	return NewWebhookExecutionStore(db)
}
