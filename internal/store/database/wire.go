// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
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
	ProvideSpaceStore,
	ProvideRepoStore,
	ProvideTokenStore,
)

// ProvideDatabase provides a database connection.
func ProvideDatabase(config *types.Config) (*sqlx.DB, error) {
	return Connect(
		config.Database.Driver,
		config.Database.Datasource,
	)
}

// ProvideUserStore provides a user store.
func ProvideUserStore(db *sqlx.DB) store.UserStore {
	switch db.DriverName() {
	case postgres:
		return NewUserStore(db)
	default:
		return NewUserStoreSync(
			NewUserStore(db),
		)
	}
}

// ProvideServiceAccountStore provides a service account store.
func ProvideServiceAccountStore(db *sqlx.DB) store.ServiceAccountStore {
	switch db.DriverName() {
	case postgres:
		return NewServiceAccountStore(db)
	default:
		return NewServiceAccountStoreSync(
			NewServiceAccountStore(db),
		)
	}
}

// ProvideSpaceStore provides a space store.
func ProvideSpaceStore(db *sqlx.DB) store.SpaceStore {
	switch db.DriverName() {
	case postgres:
		return NewSpaceStore(db)
	default:
		return NewSpaceStoreSync(
			NewSpaceStore(db),
		)
	}
}

// ProvideRepoStore provides a repo store.
func ProvideRepoStore(db *sqlx.DB) store.RepoStore {
	switch db.DriverName() {
	case postgres:
		return NewRepoStore(db)
	default:
		return NewRepoStoreSync(
			NewRepoStore(db),
		)
	}
}

// ProvideTokenStore provides a token store.
func ProvideTokenStore(db *sqlx.DB) store.TokenStore {
	switch db.DriverName() {
	case postgres:
		return NewTokenStore(db)
	default:
		return NewTokenStoreSync(
			NewTokenStore(db),
		)
	}
}
