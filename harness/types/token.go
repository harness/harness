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
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

// Represents server side infos stored for tokens we distribute.
type Token struct {
	// TODO: int64 ID doesn't match DB
	ID          int64          `db:"token_id"                 json:"-"`
	PrincipalID int64          `db:"token_principal_id"       json:"principal_id"`
	Type        enum.TokenType `db:"token_type"               json:"type"`
	Identifier  string         `db:"token_uid"                json:"identifier"`
	// ExpiresAt is an optional unix time that if specified restricts the validity of a token.
	ExpiresAt *int64 `db:"token_expires_at"         json:"expires_at,omitempty"`
	// IssuedAt is the unix time at which the token was issued.
	IssuedAt  int64 `db:"token_issued_at"          json:"issued_at"`
	CreatedBy int64 `db:"token_created_by"         json:"created_by"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (t Token) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Token
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(t),
		UID:   t.Identifier,
	})
}

// TokenResponse is returned as part of token creation for PAT / SAT / User Session.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Token       Token  `json:"token"`
}
