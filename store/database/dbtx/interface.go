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

package dbtx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Accessor is the SQLx database manipulation interface.
type Accessor interface {
	sqlx.ExtContext // sqlx.binder + sqlx.QueryerContext + sqlx.ExecerContext
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)

	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// Transaction is the Go's standard sql transaction interface.
type Transaction interface {
	Commit() error
	Rollback() error
}

type Transactor interface {
	WithTx(ctx context.Context, txFn func(ctx context.Context) error, opts ...interface{}) error
}

// AccessorTx is used to access the database. It combines Accessor interface
// with Transactor (capability to run functions in a transaction).
type AccessorTx interface {
	Accessor
	Transactor
}

// TransactionAccessor combines data access capabilities with the transaction commit and rollback.
type TransactionAccessor interface {
	Transaction
	Accessor
}
