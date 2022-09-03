// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"errors"
	"strings"

	"github.com/harness/gitness/types/enum"
)

/*
 * Splits an FQN into the parent and the leave.
 * e.g. /space1/space2/space3 -> (/space1/space2, space3, nil)
 * TODO: move to better locaion
 */
func DisectFqn(fqn string) (string, string, error) {
	if fqn == "" {
		return "", "", errors.New("Can't disect empty fqn.")
	}

	i := strings.LastIndex(fqn, "/")
	if i == -1 {
		return "", fqn, nil
	}

	return fqn[:i], fqn[i+1:], nil
}

type Space struct {
	ID          int64  `db:"space_id"              json:"id"`
	Name        string `db:"space_name"            json:"name"`
	Fqsn        string `db:"space_fqsn"            json:"fqsn"`
	ParentId    int64  `db:"space_parentId"        json:"parentId"`
	Description string `db:"space_description"     json:"description"`
	Created     int64  `db:"space_created"         json:"created"`
	Updated     int64  `db:"space_updated"         json:"updated"`
}

// Stores spaces query parameters.
type SpaceFilter struct {
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Sort  enum.SpaceAttr `json:"sort"`
	Order enum.Order     `json:"direction"`
}
