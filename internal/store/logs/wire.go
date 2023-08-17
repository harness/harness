// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideLogStore,
)

func ProvideLogStore(db *sqlx.DB, config *types.Config) store.LogStore {
	s := NewDatabaseLogStore(db)
	if config.Logs.S3.Bucket != "" {
		p := NewS3LogStore(
			config.Logs.S3.Bucket,
			config.Logs.S3.Prefix,
			config.Logs.S3.Endpoint,
			config.Logs.S3.PathStyle,
		)
		return NewCombined(p, s)
	}
	return s
}
