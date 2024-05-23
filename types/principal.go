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

import (
	"github.com/harness/gitness/types/enum"
)

// AnonymousPrincipalUID is an internal UID for anonymous principals.
const AnonymousPrincipalUID = "anonymous"

// Principal represents the identity of an acting entity (User, ServiceAccount, Service).
type Principal struct {
	// TODO: int64 ID doesn't match DB
	ID          int64              `db:"principal_id"           json:"-"`
	UID         string             `db:"principal_uid"          json:"uid"`
	Email       string             `db:"principal_email"        json:"email"`
	Type        enum.PrincipalType `db:"principal_type"         json:"type"`
	DisplayName string             `db:"principal_display_name" json:"display_name"`
	Admin       bool               `db:"principal_admin"        json:"admin"`

	// Should be part of principal or not?
	Blocked bool   `db:"principal_blocked"            json:"blocked"`
	Salt    string `db:"principal_salt"               json:"-"`

	// Other info
	Created int64 `db:"principal_created"                json:"created"`
	Updated int64 `db:"principal_updated"                json:"updated"`
}

func (p *Principal) ToPrincipalInfo() *PrincipalInfo {
	return &PrincipalInfo{
		ID:          p.ID,
		UID:         p.UID,
		DisplayName: p.DisplayName,
		Email:       p.Email,
		Type:        p.Type,
		Created:     p.Created,
		Updated:     p.Updated,
	}
}

// PrincipalInfo is a compressed representation of a principal we return as part of non-principal APIs.
type PrincipalInfo struct {
	ID          int64              `json:"id"`
	UID         string             `json:"uid"`
	DisplayName string             `json:"display_name"`
	Email       string             `json:"email"`
	Type        enum.PrincipalType `json:"type"`
	Created     int64              `json:"created"`
	Updated     int64              `json:"updated"`
}

func (p *PrincipalInfo) Identifier() int64 {
	return p.ID
}

type PrincipalFilter struct {
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
	Query string               `json:"query"`
	Types []enum.PrincipalType `json:"types"`
}
