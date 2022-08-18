// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package
var WireSet = wire.NewSet(
	ProvideDatabase,
	ProvideUserStore,
	ProvidePipelineStore,
	ProvideExecutionStore,
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
	case "postgres":
		return NewUserStore(db)
	default:
		return NewUserStoreSync(
			NewUserStore(db),
		)
	}
}

// ProvidePipelineStore provides a pipeline store.
func ProvidePipelineStore(db *sqlx.DB) store.PipelineStore {
	switch db.DriverName() {
	case "postgres":
		return NewPipelineStore(db)
	default:
		return NewPipelineStoreSync(
			NewPipelineStore(db),
		)
	}
}

// ProvideExecutionStore provides a execution store.
func ProvideExecutionStore(db *sqlx.DB) store.ExecutionStore {
	switch db.DriverName() {
	case "postgres":
		return NewExecutionStore(db)
	default:
		return NewExecutionStoreSync(
			NewExecutionStore(db),
		)
	}
}
