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

package auth

import (
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/types/enum"
)

type Metadata interface {
	ImpactsAuthorization() bool
}

// EmptyMetadata represents the state when the auth session doesn't have any extra metadata.
type EmptyMetadata struct{}

func (m *EmptyMetadata) ImpactsAuthorization() bool {
	return false
}

// TokenMetadata contains information about the token that was used during auth.
type TokenMetadata struct {
	TokenType enum.TokenType
	TokenID   int64
}

func (m *TokenMetadata) ImpactsAuthorization() bool {
	return false
}

// MembershipMetadata contains information about an ephemeral membership grant.
type MembershipMetadata struct {
	SpaceID int64
	Role    enum.MembershipRole
}

func (m *MembershipMetadata) ImpactsAuthorization() bool {
	return true
}

// AccessPermissionMetadata contains information about permissions per space.
type AccessPermissionMetadata struct {
	AccessPermissions *jwt.SubClaimsAccessPermissions
}

func (m *AccessPermissionMetadata) ImpactsAuthorization() bool {
	return true
}
