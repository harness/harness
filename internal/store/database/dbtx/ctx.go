// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package dbtx

import (
	"context"
	"github.com/jmoiron/sqlx"
)

// ctxKeyDB is context key for storing and retrieving Transactor to and from a context.
type ctxKeyDB struct{}

// PutTransactor places Transactor into the context.
func PutTransactor(ctx context.Context, t Transactor) context.Context {
	return context.WithValue(ctx, ctxKeyDB{}, t)
}

// WithTx starts a transaction with Transactor interface from the context. It will panic if there is no Transactor.
func WithTx(ctx context.Context, txFn func(ctx context.Context) error, opts ...interface{}) error {
	return ctx.Value(ctxKeyDB{}).(Transactor).WithTx(ctx, txFn, opts...)
}

// ctxKeyTx is context key for storing and retrieving Tx to and from a context.
type ctxKeyTx struct{}

// GetAccessor returns Accessor interface from the context if it exists or creates a new one from the provided *sql.DB.
// It is intended to be used in data layer functions that might or might not be running inside a transaction.
func GetAccessor(ctx context.Context, db *sqlx.DB) Accessor {
	if a, ok := ctx.Value(ctxKeyTx{}).(Accessor); ok {
		return a
	}
	return db
}

// GetTransaction returns Transaction interface from the context if it exists or return nil.
// It is intended to be used in transactions in service layer functions to explicitly commit or rollback transactions.
func GetTransaction(ctx context.Context) Transaction {
	if a, ok := ctx.Value(ctxKeyTx{}).(Transaction); ok {
		return a
	}
	return nil
}
