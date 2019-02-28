// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build oss

package db

import (
	"database/sql"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/drone/drone/store/shared/migrate/sqlite"
)

// Connect to an embedded sqlite database.
func Connect(driver, datasource string) (*DB, error) {
	db, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, err
	}
	if err := sqlite.Migrate(db); err != nil {
		return nil, err
	}
	return &DB{
		conn:   sqlx.NewDb(db, driver),
		lock:   &sync.RWMutex{},
		driver: Sqlite,
	}, nil
}
