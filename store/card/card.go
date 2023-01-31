// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.
// +build !oss

package card

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new card database store.
func New(db *db.DB) core.CardStore {
	return &cardStore{
		db: db,
	}
}

type cardStore struct {
	db *db.DB
}

type card struct {
	Id   int64  `json:"id,omitempty"`
	Data []byte `json:"card_data"`
}

func (c cardStore) Find(ctx context.Context, step int64) (io.ReadCloser, error) {
	out := &card{Id: step}
	err := c.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params, err := toParams(out)
		if err != nil {
			return err
		}
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})

	return ioutil.NopCloser(
		bytes.NewBuffer(out.Data),
	), err
}

func (c cardStore) Create(ctx context.Context, step int64, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		in := &card{
			Id:   step,
			Data: data,
		}
		params, err := toParams(in)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

func (c *cardStore) Update(ctx context.Context, step int64, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		card := &card{
			Id:   step,
			Data: data,
		}
		params, err := toParams(card)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtUpdate, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

func (c cardStore) Delete(ctx context.Context, step int64) error {
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := &card{
			Id: step,
		}
		stmt, args, err := binder.BindNamed(stmtDelete, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

const queryBase = `
SELECT
 card_id
,card_data
`

const queryKey = queryBase + `
FROM cards
WHERE card_id = :card_id
LIMIT 1
`

const stmtInsert = `
INSERT INTO cards (
 card_id
,card_data
) VALUES (
 :card_id
,:card_data
)
`

const stmtUpdate = `
UPDATE cards
SET card_data = :card_data
WHERE card_id = :card_id
`

const stmtDelete = `
DELETE FROM cards
WHERE card_id = :card_id
`

const stmtInsertPostgres = stmtInsert + `
RETURNING card_id
`
