// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package card

import (
	"github.com/drone/drone/store/shared/db"
)

// helper function converts the card structure to a set
// of named query parameters.
func toParams(card *card) (map[string]interface{}, error) {
	return map[string]interface{}{
		"card_id":   card.Id,
		"card_data": card.Data,
	}, nil
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(scanner db.Scanner, dst *card) error {
	err := scanner.Scan(
		&dst.Id,
		&dst.Data,
	)
	if err != nil {
		return err
	}
	return nil
}
