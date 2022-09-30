// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import "github.com/harness/gitness/types/enum"

type (
	// ServiceAccount is a principal representing a service account.
	ServiceAccount struct {
		// Fields from Principal (without admin, as it's never an admin)
		ID      int64  `db:"principal_id"          json:"-"`
		UID     string `db:"principal_uid"         json:"uid"`
		Name    string `db:"principal_name"        json:"name"`
		Blocked bool   `db:"principal_blocked"     json:"blocked"`
		Salt    string `db:"principal_salt"        json:"-"`
		Created int64  `db:"principal_created"     json:"created"`
		Updated int64  `db:"principal_updated"     json:"updated"`

		// ServiceAccount specific fields
		ParentType enum.ParentResourceType `db:"principal_sa_parentType"  json:"parentType"`
		ParentID   int64                   `db:"principal_sa_parentId"    json:"parentId"`
	}

	// ServiceAccountInput store details used to
	// create or update a service account.
	ServiceAccountInput struct {
		Name       *string                  `json:"name"`
		ParentType *enum.ParentResourceType `json:"parentType"`
		ParentID   *int64                   `json:"parentId"`
	}
)
