// Copyright 2022 Harness Inc. All rights reserved.
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
		ID          int64  `db:"principal_id"             json:"-"`
		UID         string `db:"principal_uid"            json:"uid"`
		Email       string `db:"principal_email"          json:"email"`
		DisplayName string `db:"principal_display_name"   json:"display_name"`
		Admin       bool   `db:"principal_admin"          json:"admin"`
		Blocked     bool   `db:"principal_blocked"        json:"blocked"`
		Salt        string `db:"principal_salt"           json:"-"`
		Created     int64  `db:"principal_created"        json:"created"`
		Updated     int64  `db:"principal_updated"        json:"updated"`

		// User specific fields
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
		Order enum.Order    `json:"order"`
	}
)

func (u *User) ToPrincipal() *Principal {
	return &Principal{
		ID:          u.ID,
		UID:         u.UID,
		Email:       u.Email,
		Type:        enum.PrincipalTypeUser,
		DisplayName: u.DisplayName,
		Admin:       u.Admin,
		Blocked:     u.Blocked,
		Salt:        u.Salt,
		Created:     u.Created,
		Updated:     u.Updated,
	}
}

func (u *User) ToPrincipalInfo() *PrincipalInfo {
	return u.ToPrincipal().ToPrincipalInfo()
}
