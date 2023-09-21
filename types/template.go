// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Template struct {
	ID          int64  `db:"template_id"              json:"id"`
	Description string `db:"template_description"     json:"description"`
	SpaceID     int64  `db:"template_space_id"        json:"space_id"`
	UID         string `db:"template_uid"             json:"uid"`
	Data        string `db:"template_data"            json:"data"`
	Created     int64  `db:"template_created"         json:"created"`
	Updated     int64  `db:"template_updated"         json:"updated"`
	Version     int64  `db:"template_version"         json:"-"`
}
