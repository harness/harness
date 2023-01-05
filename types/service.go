// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// Service is a principal representing a different internal service that runs alongside gitness.
	Service struct {
		// Fields from Principal
		ID          int64  `db:"principal_id"           json:"-"`
		UID         string `db:"principal_uid"          json:"uid"`
		Email       string `db:"principal_email"        json:"email"`
		DisplayName string `db:"principal_display_name" json:"display_name"`
		Admin       bool   `db:"principal_admin"        json:"admin"`
		Blocked     bool   `db:"principal_blocked"      json:"blocked"`
		Salt        string `db:"principal_salt"         json:"-"`
		Created     int64  `db:"principal_created"      json:"created"`
		Updated     int64  `db:"principal_updated"      json:"updated"`
	}
)

func (s *Service) ToPrincipal() *Principal {
	return &Principal{
		ID:          s.ID,
		UID:         s.UID,
		Email:       s.Email,
		Type:        enum.PrincipalTypeService,
		DisplayName: s.DisplayName,
		Admin:       s.Admin,
		Blocked:     s.Blocked,
		Salt:        s.Salt,
		Created:     s.Created,
		Updated:     s.Updated,
	}
}

func (s *Service) ToPrincipalInfo() *PrincipalInfo {
	return &PrincipalInfo{
		ID:          s.ID,
		UID:         s.UID,
		DisplayName: s.DisplayName,
		Email:       s.Email,
	}
}
