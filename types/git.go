// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

const NilSHA = "0000000000000000000000000000000000000000"

// CommitFilter stores commit query parameters.
type CommitFilter struct {
	After string `json:"after"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

// BranchFilter stores branch query parameters.
type BranchFilter struct {
	Query string                `json:"query"`
	Sort  enum.BranchSortOption `json:"sort"`
	Order enum.Order            `json:"order"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// TagFilter stores commit tag query parameters.
type TagFilter struct {
	Query string             `json:"query"`
	Sort  enum.TagSortOption `json:"sort"`
	Order enum.Order         `json:"order"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}
