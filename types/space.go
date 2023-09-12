// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

/*
Space represents a space.
There isn't a one-solves-all hierarchical data structure for DBs,
so for now we are using a mix of materialized paths and adjacency list.
Every space stores its parent, and a space's path is stored in a separate table.
PRO: Quick lookup of childs, quick lookup based on fqdn (apis)
CON: Changing a space uid requires changing all its ancestors' Paths.

Interesting reads:
https://stackoverflow.com/questions/4048151/what-are-the-options-for-storing-hierarchical-data-in-a-relational-database
https://www.slideshare.net/billkarwin/models-for-hierarchical-data
*/
type Space struct {
	ID          int64  `json:"id"`
	Version     int64  `json:"-"`
	ParentID    int64  `json:"parent_id"`
	Path        string `json:"path"`
	UID         string `json:"uid"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	CreatedBy   int64  `json:"created_by"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
}

// Stores spaces query parameters.
type SpaceFilter struct {
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Query string         `json:"query"`
	Sort  enum.SpaceAttr `json:"sort"`
	Order enum.Order     `json:"order"`
}
