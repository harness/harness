// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package dbtx

import (
	"context"
	"database/sql"
	"errors"
)

type runnerDB struct {
	transactor
}

var _ Transactor = runnerDB{}

func (r runnerDB) WithTx(ctx context.Context, txFn func(context.Context) error, opts ...interface{}) error {
	var txOpts *sql.TxOptions
	for _, opt := range opts {
		if v, ok := opt.(*sql.TxOptions); ok {
			txOpts = v
		}
	}

	if txOpts == nil {
		txOpts = TxDefault
	}

	tx, err := r.startTx(ctx, txOpts)
	if err != nil {
		return err
	}

	rtx := &runnerTx{
		Tx:       tx,
		commit:   false,
		rollback: false,
	}

	defer func() {
		if rtx.commit || rtx.rollback {
			return
		}
		_ = tx.Rollback() // ignoring the rollback error
	}()

	err = txFn(context.WithValue(ctx, ctxKeyTx{}, Tx(rtx)))
	if err != nil {
		return err
	}

	if !rtx.commit && !rtx.rollback {
		err = rtx.Commit()
		if errors.Is(err, sql.ErrTxDone) {
			// Check if the transaction failed because of the context, if yes return the ctx error.
			if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
				err = ctxErr
			}
		}
	}

	return err
}

type runnerTx struct {
	Tx
	commit   bool
	rollback bool
}

var _ Tx = (*runnerTx)(nil)

func (r *runnerTx) Commit() error {
	err := r.Tx.Commit()
	if err == nil {
		r.commit = true
	}
	return err
}

func (r *runnerTx) Rollback() error {
	err := r.Tx.Rollback()
	if err == nil {
		r.rollback = true
	}
	return err
}
