// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Connector struct {
	ID          int64  `db:"connector_id"              json:"id"`
	Description string `db:"connector_description"     json:"description"`
	SpaceID     int64  `db:"connector_space_id"        json:"space_id"`
	UID         string `db:"connector_uid"             json:"uid"`
	Type        string `db:"connector_type"             json:"type"`
	Data        string `db:"connector_data"            json:"data"`
	Created     int64  `db:"connector_created"         json:"created"`
	Updated     int64  `db:"connector_updated"         json:"updated"`
	Version     int64  `db:"connector_version"         json:"-"`
}
