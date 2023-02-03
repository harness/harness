// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// ServiceAccount is a principal representing a service account.
	ServiceAccount struct {
		// Fields from Principal (without admin, as it's never an admin)
		ID          int64  `db:"principal_id"           json:"-"`
		UID         string `db:"principal_uid"          json:"uid"`
		Email       string `db:"principal_email"        json:"email"`
		DisplayName string `db:"principal_display_name" json:"display_name"`
		Admin       bool   `db:"principal_admin"        json:"admin"`
		Blocked     bool   `db:"principal_blocked"      json:"blocked"`
		Salt        string `db:"principal_salt"         json:"-"`
		Created     int64  `db:"principal_created"      json:"created"`
		Updated     int64  `db:"principal_updated"      json:"updated"`

		// ServiceAccount specific fields
		ParentType enum.ParentResourceType `db:"principal_sa_parent_type"  json:"parent_type"`
		ParentID   int64                   `db:"principal_sa_parent_id"    json:"parent_id"`
	}

	// ServiceAccountInput store details used to
	// create or update a service account.
	ServiceAccountInput struct {
		DisplayName *string                  `json:"display_name"`
		ParentType  *enum.ParentResourceType `json:"parent_type"`
		ParentID    *int64                   `json:"parent_id"`
	}
)

func (s *ServiceAccount) ToPrincipal() *Principal {
	return &Principal{
		ID:          s.ID,
		UID:         s.UID,
		Email:       s.Email,
		Type:        enum.PrincipalTypeServiceAccount,
		DisplayName: s.DisplayName,
		Admin:       s.Admin,
		Blocked:     s.Blocked,
		Salt:        s.Salt,
		Created:     s.Created,
		Updated:     s.Updated,
	}
}

func (s *ServiceAccount) ToPrincipalInfo() *PrincipalInfo {
	return s.ToPrincipal().ToPrincipalInfo()
}
