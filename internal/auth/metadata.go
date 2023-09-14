// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import "github.com/harness/gitness/types/enum"

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
