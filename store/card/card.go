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

func (c *cardStore) FindByBuild(ctx context.Context, build int64) ([]*core.Card, error) {
	var out []*core.Card
	err := c.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"card_build": build,
		}
		stmt, args, err := binder.BindNamed(queryByBuild, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (c cardStore) Find(ctx context.Context, step int64) (*core.Card, error) {
	out := &core.Card{Step: step}
	err := c.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params, err := toParams(out)
		if err != nil {
			return err
		}
		query, args, err := binder.BindNamed(queryByStep, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (c cardStore) FindData(ctx context.Context, id int64) (io.ReadCloser, error) {
	out := &core.CardData{}
	err := c.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"card_id": id,
		}
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRowCardDataOnly(row, out)
	})
	return ioutil.NopCloser(
		bytes.NewBuffer(out.Data),
	), err
}

func (c cardStore) Create(ctx context.Context, card *core.Card, data io.ReadCloser) error {
	if c.db.Driver() == db.Postgres {
		return c.createPostgres(ctx, card, data)
	}
	return c.create(ctx, card, data)
}

func (c *cardStore) create(ctx context.Context, card *core.Card, data io.ReadCloser) error {
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		cardData, err := ioutil.ReadAll(data)
		if err != nil {
			return err
		}
		params, err := toSaveCardParams(card, cardData)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		card.Id, err = res.LastInsertId()
		return err
	})
}

func (c *cardStore) createPostgres(ctx context.Context, card *core.Card, data io.ReadCloser) error {
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		cardData, err := ioutil.ReadAll(data)
		if err != nil {
			return err
		}
		params, err := toSaveCardParams(card, cardData)
		if err != nil {
			return err
		}
		stmt, args, err := binder.BindNamed(stmtInsertPostgres, params)
		if err != nil {
			return err
		}
		return execer.QueryRow(stmt, args...).Scan(&card.Id)
	})
}

func (c cardStore) Delete(ctx context.Context, id int64) error {
	return c.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := map[string]interface{}{
			"card_id": id,
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
,card_build
,card_stage
,card_step
,card_schema
`

const queryCardData = `
SELECT
 card_id
,card_data
`

const queryByBuild = queryBase + `
FROM cards
WHERE card_build = :card_build
`

const queryByStep = queryBase + `
FROM cards
WHERE card_step = :card_step
LIMIT 1
`

const queryKey = queryCardData + `
FROM cards
WHERE card_id = :card_id
LIMIT 1
`

const stmtInsert = `
INSERT INTO cards (
 card_build
,card_stage
,card_step
,card_schema
,card_data
) VALUES (
 :card_build
,:card_stage
,:card_step
,:card_schema
,:card_data
)
`

const stmtDelete = `
DELETE FROM cards
WHERE card_id = :card_id
`

const stmtInsertPostgres = stmtInsert + `
RETURNING card_id
`
