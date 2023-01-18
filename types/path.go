// Copyright 2022 Harness Inc. All rights reserved.
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
	// TODO: int64 ID doesn't match DB
	ID         int64               `json:"id"`
	Version    int64               `json:"-"`
	Value      string              `json:"value"`
	IsPrimary  bool                `json:"is_primary"`
	TargetType enum.PathTargetType `json:"target_type"`
	TargetID   int64               `json:"target_id"`
	CreatedBy  int64               `json:"created_by"`
	Created    int64               `json:"created"`
	Updated    int64               `json:"updated"`
}

func (p *Path) Identifier() string {
	return p.Value
}

// PathParams used for creating paths (alias or rename).
type PathParams struct {
	Path      string
	CreatedBy int64
	Created   int64
	Updated   int64
}

// PathFilter stores path query parameters.
type PathFilter struct {
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Sort  enum.PathAttr `json:"sort"`
	Order enum.Order    `json:"order"`
}
