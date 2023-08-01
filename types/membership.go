// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// MembershipKey can be used as a key for finding a user's space membership info.
type MembershipKey struct {
	SpaceID     int64
	PrincipalID int64
}

// Membership represents a user's membership of a space.
type Membership struct {
	MembershipKey `json:"-"`

	CreatedBy int64 `json:"-"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`

	Role enum.MembershipRole `json:"role"`
}

// MembershipUser adds user info to the Membership data.
type MembershipUser struct {
	Membership
	Principal PrincipalInfo `json:"principal"`
	AddedBy   PrincipalInfo `json:"added_by"`
}

// MembershipSpace adds space info to the Membership data.
type MembershipSpace struct {
	Membership
	Space   Space         `json:"space"`
	AddedBy PrincipalInfo `json:"added_by"`
}

// MembershipFilter holds membership query parameters.
type MembershipFilter struct {
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
	Query string              `json:"query"`
	Sort  enum.MembershipSort `json:"sort"`
	Order enum.Order          `json:"order"`
}
