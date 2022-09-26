// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// Represents the identity of an acting entity (User, ServiceAccount, Service).
	Principal struct {
		ID    int64              `db:"principal_id"          json:"id"`
		Type  enum.PrincipalType `db:"principal_type"        json:"type"`
		Name  string             `db:"principal_name"        json:"name"`
		Admin bool               `db:"principal_admin"       json:"admin"`

		// Should be part of principal or not?
		ExternalID string `db:"principal_externalId"         json:"externalId"`
		Blocked    bool   `db:"principal_blocked"            json:"blocked"`
		Salt       string `db:"principal_salt"               json:"-"`

		// Other info
		Created int64 `db:"principal_created"                json:"created"`
		Updated int64 `db:"principal_updated"                json:"updated"`
	}
)

func PrincipalFromUser(user *User) *Principal {
	return &Principal{
		ID:         user.ID,
		Type:       enum.PrincipalTypeUser,
		Name:       user.Name,
		Admin:      user.Admin,
		ExternalID: user.ExternalID,
		Blocked:    user.Blocked,
		Salt:       user.Salt,
		Created:    user.Created,
		Updated:    user.Updated,
	}
}

func PrincipalFromServiceAccount(sa *ServiceAccount) *Principal {
	return &Principal{
		ID:         sa.ID,
		Type:       enum.PrincipalTypeServiceAccount,
		Name:       sa.Name,
		Admin:      false,
		ExternalID: sa.ExternalID,
		Blocked:    sa.Blocked,
		Salt:       sa.Salt,
		Created:    sa.Created,
		Updated:    sa.Updated,
	}
}
