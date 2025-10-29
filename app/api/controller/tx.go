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

package controller

import (
	"context"
	"errors"

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
)

// TxOptionRetryCount transaction option allows setting number of transaction executions reties.
// A transaction started with TxOptLock will be automatically retried in case of version conflict error.
type TxOptionRetryCount int

// TxOptionResetFunc transaction provides a function that will be executed before the transaction retry.
// A transaction started with TxOptLock will be automatically retried in case of version conflict error.
type TxOptionResetFunc func()

// TxOptLock runs the provided function inside a database transaction. If optimistic lock error occurs
// during the operation, the function will retry the whole transaction again (to the maximum of 5 times,
// but this can be overridden by providing an additional TxOptionRetryCount option).
func TxOptLock(ctx context.Context,
	tx dbtx.Transactor,
	txFn func(ctx context.Context) error,
	opts ...any,
) (err error) {
	tries := 5
	var resetFuncs []func()
	for _, opt := range opts {
		if n, ok := opt.(TxOptionRetryCount); ok {
			tries = int(n)
		}
		if fn, ok := opt.(TxOptionResetFunc); ok {
			resetFuncs = append(resetFuncs, fn)
		}
	}

	for try := 0; try < tries; try++ {
		err = tx.WithTx(ctx, txFn, opts...)
		if !errors.Is(err, store.ErrVersionConflict) {
			break
		}

		for _, fn := range resetFuncs {
			fn()
		}
	}

	return
}
