// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

const (
	PathSeparator = "/"
)

// SpacePath represents a full path to a space.
type SpacePath struct {
	Value     string `json:"value"`
	IsPrimary bool   `json:"is_primary"`
	SpaceID   int64  `json:"space_id"`
}

// SpacePathSegment represents a segment of a path to a space.
type SpacePathSegment struct {
	// TODO: int64 ID doesn't match DB
	ID        int64  `json:"id"`
	UID       string `json:"uid"`
	IsPrimary bool   `json:"is_primary"`
	SpaceID   int64  `json:"space_id"`
	ParentID  int64  `json:"parent_id"`
	CreatedBy int64  `json:"created_by"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
}
