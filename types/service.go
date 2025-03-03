// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// Service is a principal representing a different internal service that runs alongside gitness.
	Service struct {
		// Fields from Principal
		ID          int64  `db:"principal_id"           json:"id"`
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
	return s.ToPrincipal().ToPrincipalInfo()
}
