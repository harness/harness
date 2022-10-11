// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

type (
	// Service is a principal representing a different internal service that runs alongside gitness.
	Service struct {
		// Fields from Principal
		ID      int64  `db:"principal_id"          json:"-"`
		UID     string `db:"principal_uid"         json:"uid"`
		Name    string `db:"principal_name"        json:"name"`
		Admin   bool   `db:"principal_admin"       json:"admin"`
		Blocked bool   `db:"principal_blocked"     json:"blocked"`
		Salt    string `db:"principal_salt"        json:"-"`
		Created int64  `db:"principal_created"     json:"created"`
		Updated int64  `db:"principal_updated"     json:"updated"`
	}
)
