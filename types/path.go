// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

const (
	PathSeparator = "/"
)

// Represents a path to a resource (e.g. space) that can be used to address the resource.
type Path struct {
	ID         int64               `db:"path_id"              json:"id"`
	Value      string              `db:"path_value"           json:"value"`
	IsAlias    bool                `db:"path_isAlias"         json:"isAlias"`
	TargetType enum.PathTargetType `db:"path_targetType"      json:"targetType"`
	TargetId   int64               `db:"path_targetId"        json:"targetId"`
	CreatedBy  int64               `db:"path_createdBy"       json:"createdBy"`
	Created    int64               `db:"path_created"         json:"created"`
	Updated    int64               `db:"path_updated"         json:"updated"`
}

// Used for creating paths (alias or rename)
type PathParams struct {
	Path      string
	CreatedBy int64
	Created   int64
	Updated   int64
}

// Stores path query parameters.
type PathFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Sort  enum.PathAttr `json:"sort"`
	Order enum.Order    `json:"direction"`
}
