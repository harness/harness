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

// MembershipUserFilter holds membership user query parameters.
type MembershipUserFilter struct {
	ListQueryFilter
	Sort  enum.MembershipUserSort `json:"sort"`
	Order enum.Order              `json:"order"`
}

// MembershipSpace adds space info to the Membership data.
type MembershipSpace struct {
	Membership
	Space   Space         `json:"space"`
	AddedBy PrincipalInfo `json:"added_by"`
}

// MembershipSpaceFilter holds membership space query parameters.
type MembershipSpaceFilter struct {
	ListQueryFilter
	Sort  enum.MembershipSpaceSort `json:"sort"`
	Order enum.Order               `json:"order"`
}
