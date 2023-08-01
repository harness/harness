// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// Represents server side infos stored for tokens we distribute.
type Token struct {
	// TODO: int64 ID doesn't match DB
	ID          int64          `db:"token_id"                 json:"-"`
	PrincipalID int64          `db:"token_principal_id"       json:"principal_id"`
	Type        enum.TokenType `db:"token_type"               json:"type"`
	UID         string         `db:"token_uid"                json:"uid"`
	// ExpiresAt is an optional unix time that if specified restricts the validity of a token.
	ExpiresAt *int64 `db:"token_expires_at"         json:"expires_at,omitempty"`
	// IssuedAt is the unix time at which the token was issued.
	IssuedAt  int64            `db:"token_issued_at"          json:"issued_at"`
	Grants    enum.AccessGrant `db:"token_grants"             json:"grants"`
	CreatedBy int64            `db:"token_created_by"         json:"created_by"`
}

// TokenResponse is returned as part of token creation for PAT / SAT / User Session.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Token       Token  `json:"token"`
}
