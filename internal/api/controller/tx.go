// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
