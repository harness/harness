// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"github.com/harness/gitness/app/store"
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
