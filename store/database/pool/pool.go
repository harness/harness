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

// Package pool sizes the connection pool used by the database handle.
//
// The Connect helper opens *sqlx.DB without setting any pool bounds, leaving
// Go's database/sql defaults (MaxOpenConns=0 == unlimited, MaxIdleConns=2).
// Under bursty traffic an unbounded pool exhausts Postgres max_connections and
// surfaces as 5xx to clients. Apply sets the bounds supplied by the config.
//
// The values always come from the service config, which owns the defaults
// (see types.Config.Database). This package therefore does not carry its own
// defaults; it validates the supplied config and errors on unset values so a
// misconfiguration fails fast at startup instead of silently falling back.
//
// The setters are driver-agnostic: database/sql applies them the same way for
// both postgres (lib/pq) and sqlite3 (mattn/go-sqlite3), so this is safe to run
// regardless of the configured driver.
package pool

import (
	"fmt"
	"time"
)

// Config controls *sql.DB pool sizing. All values are required and must be
// positive; they are supplied from the service config, which defines the
// defaults via envconfig.
type Config struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type DB interface {
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	SetConnMaxLifetime(d time.Duration)
}

// Apply sizes the pool from cfg. It returns an error if any value is unset
// (<= 0), since valid values are expected to come from the service config.
func Apply(db DB, cfg Config) error {
	if cfg.MaxOpenConns <= 0 {
		return fmt.Errorf("database pool: MaxOpenConns must be > 0, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns <= 0 {
		return fmt.Errorf("database pool: MaxIdleConns must be > 0, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime <= 0 {
		return fmt.Errorf("database pool: ConnMaxLifetime must be > 0, got %s", cfg.ConnMaxLifetime)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return nil
}
