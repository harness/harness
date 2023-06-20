// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package controller

import (
	"context"
	"errors"

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type TxOptionRetryCount int

// TxOptLock runs the provided function inside a database transaction. If optimistic lock error occurs
// during the operation, the function will retry the whole transaction again (to the maximum of 5 times,
// but this can be overridden by providing an additional TxOptionRetryCount option).
func TxOptLock(ctx context.Context,
	db *sqlx.DB,
	txFn func(ctx context.Context) error,
	opts ...interface{},
) (err error) {
	tries := 5
	for _, opt := range opts {
		if n, ok := opt.(TxOptionRetryCount); ok {
			tries = int(n)
		}
	}

	for try := 0; try < tries; try++ {
		err = dbtx.New(db).WithTx(ctx, txFn, opts...)
		if !errors.Is(err, store.ErrVersionConflict) {
			break
		}
	}

	return
}
