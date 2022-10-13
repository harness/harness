// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// Represents the identity of an acting entity (User, ServiceAccount, Service).
	Principal struct {
		// TODO: int64 ID doesn't match DB
		ID    int64              `db:"principal_id"          json:"-"`
		UID   string             `db:"principal_uid"         json:"uid"`
		Type  enum.PrincipalType `db:"principal_type"        json:"type"`
		Name  string             `db:"principal_name"        json:"name"`
		Admin bool               `db:"principal_admin"       json:"admin"`

		// Should be part of principal or not?
		Blocked bool   `db:"principal_blocked"            json:"blocked"`
		Salt    string `db:"principal_salt"               json:"-"`

		// Other info
		Created int64 `db:"principal_created"                json:"created"`
		Updated int64 `db:"principal_updated"                json:"updated"`
	}
)

func PrincipalFromUser(user *User) *Principal {
	return &Principal{
		ID:      user.ID,
		UID:     user.UID,
		Type:    enum.PrincipalTypeUser,
		Name:    user.Name,
		Admin:   user.Admin,
		Blocked: user.Blocked,
		Salt:    user.Salt,
		Created: user.Created,
		Updated: user.Updated,
	}
}

func PrincipalFromServiceAccount(sa *ServiceAccount) *Principal {
	return &Principal{
		ID:      sa.ID,
		UID:     sa.UID,
		Type:    enum.PrincipalTypeServiceAccount,
		Name:    sa.Name,
		Admin:   false,
		Blocked: sa.Blocked,
		Salt:    sa.Salt,
		Created: sa.Created,
		Updated: sa.Updated,
	}
}

func PrincipalFromService(s *Service) *Principal {
	return &Principal{
		ID:      s.ID,
		UID:     s.UID,
		Type:    enum.PrincipalTypeService,
		Name:    s.Name,
		Admin:   s.Admin,
		Blocked: s.Blocked,
		Salt:    s.Salt,
		Created: s.Created,
		Updated: s.Updated,
	}
}
