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

/*
 * Represents a space.
 * There isn't a one-solves-all hierarchical data structure for DBs,
 * so for now we are using a mix of materialized paths and adjacency list,
 * meaning any space stores its full qualified space name as well as the id of its parent.
 * 	PRO: Quick lookup of childs, quick lookup based on fqdn (apis)
 *  CON: Changing a space name requires changing all its ancestors' FQNs.
 *
 * Interesting reads:
 *    https://stackoverflow.com/questions/4048151/what-are-the-options-for-storing-hierarchical-data-in-a-relational-database
 *	  https://www.slideshare.net/billkarwin/models-for-hierarchical-data
 */
type Space struct {
	ID          int64  `db:"space_id"              json:"id"`
	Name        string `db:"space_name"            json:"name"`
	Fqn         string `db:"space_fqn"             json:"fqn"`
	ParentId    int64  `db:"space_parentId"        json:"parentId"`
	DisplayName string `db:"space_displayName"     json:"displayName"`
	Description string `db:"space_description"     json:"description"`
	IsPublic    bool   `db:"space_isPublic"        json:"isPublic"`
	CreatedBy   int64  `db:"space_createdBy"       json:"createdBy"`
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
