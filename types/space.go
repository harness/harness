// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"github.com/harness/gitness/types/enum"
)

/*
Space represents a space.
There isn't a one-solves-all hierarchical data structure for DBs,
so for now we are using a mix of materialized paths and adjacency list.
Every space stores its parent, and a space's path (and aliases) is stored in a separate table.
PRO: Quick lookup of childs, quick lookup based on fqdn (apis).
CON: we require a separate table.

Interesting reads:
https://stackoverflow.com/questions/4048151/what-are-the-options-for-storing-hierarchical-data-in-a-relational-database
https://www.slideshare.net/billkarwin/models-for-hierarchical-data
*/
type Space struct {
	ID          int64  `json:"id"`
	Version     int64  `json:"-"`
	ParentID    int64  `json:"parent_id"`
	Path        string `json:"path"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
	CreatedBy   int64  `json:"created_by"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
	Deleted     *int64 `json:"deleted,omitempty"`
}

// Stores spaces query parameters.
type SpaceFilter struct {
	Page              int            `json:"page"`
	Size              int            `json:"size"`
	Query             string         `json:"query"`
	Sort              enum.SpaceAttr `json:"sort"`
	Order             enum.Order     `json:"order"`
	DeletedAt         *int64         `json:"deleted_at,omitempty"`
	DeletedBeforeOrAt *int64         `json:"deleted_before_or_at,omitempty"`
	Recursive         bool           `json:"recursive"`
}
