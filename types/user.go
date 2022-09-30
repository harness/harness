// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import (
	"github.com/harness/gitness/types/enum"
)

type (
	// User is a principal representing an end user.
	User struct {
		// Fields from Principal
		ID      int64  `db:"principal_id"            json:"-"`
		UID     string `db:"principal_uid"           json:"uid"`
		Name    string `db:"principal_name"          json:"name"`
		Admin   bool   `db:"principal_admin"         json:"admin"`
		Blocked bool   `db:"principal_blocked"       json:"blocked"`
		Salt    string `db:"principal_salt"          json:"-"`
		Created int64  `db:"principal_created"       json:"created"`
		Updated int64  `db:"principal_updated"       json:"updated"`

		// User specific fields
		Email    string `db:"principal_user_email"       json:"email"`
		Password string `db:"principal_user_password"    json:"-"`
	}

	// UserInput store user account details used to
	// create or update a user.
	UserInput struct {
		Email    *string `json:"email"`
		Password *string `json:"password"`
		Name     *string `json:"name"`
		Admin    *bool   `json:"admin"`
	}

	// UserFilter stores user query parameters.
	UserFilter struct {
		Page  int           `json:"page"`
		Size  int           `json:"size"`
		Sort  enum.UserAttr `json:"sort"`
		Order enum.Order    `json:"direction"`
	}
)
