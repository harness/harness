// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package card

import (
	"database/sql"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// helper function converts the card structure to a set
// of named query parameters.
func toParams(card *core.Card) (map[string]interface{}, error) {
	return map[string]interface{}{
		"card_id":     card.Id,
		"card_build":  card.Build,
		"card_stage":  card.Stage,
		"card_step":   card.Step,
		"card_schema": card.Schema,
	}, nil
}

// helper function converts the card structure to a set
// of named query parameters.
func toSaveCardParams(card *core.CreateCard) (map[string]interface{}, error) {
	return map[string]interface{}{
		"card_id":     card.Id,
		"card_build":  card.Build,
		"card_stage":  card.Stage,
		"card_step":   card.Step,
		"card_schema": card.Schema,
		"card_data":   card.Data,
	}, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(scanner db.Scanner, dst *core.Card) error {
	err := scanner.Scan(
		&dst.Id,
		&dst.Build,
		&dst.Stage,
		&dst.Step,
		&dst.Schema,
	)
	if err != nil {
		return err
	}
	return nil
}

func scanRowCardDataOnly(scanner db.Scanner, dst *core.CardData) error {
	return scanner.Scan(
		&dst.Id,
		&dst.Data,
	)
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRows(rows *sql.Rows) ([]*core.Card, error) {
	defer rows.Close()

	card := []*core.Card{}
	for rows.Next() {
		tem := new(core.Card)
		err := scanRow(rows, tem)
		if err != nil {
			return nil, err
		}
		card = append(card, tem)
	}
	return card, nil
}
