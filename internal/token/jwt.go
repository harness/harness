// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

const (
	issuer = "Gitness"
)

// JWTClaims defines custom token claims.
type JWTClaims struct {
	jwt.StandardClaims

	TokenType   enum.TokenType `json:"ttp,omitempty"`
	TokenID     int64          `json:"tid,omitempty"`
	PrincipalID int64          `json:"pid,omitempty"`
}

// GenerateJWTForToken generates a jwt for a given token.
func GenerateJWTForToken(token *types.Token, secret string) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		jwt.StandardClaims{
			Issuer: issuer,
			// times required to be in sec not millisec
			IssuedAt:  token.IssuedAt / 1000,
			ExpiresAt: token.ExpiresAt / 1000,
		},
		token.Type,
		token.ID,
		token.PrincipalID,
	})

	res, err := jwtToken.SignedString([]byte(secret))
	if err != nil {
		return "", errors.Wrap(err, "Failed to sign token")
	}

	return res, nil
}
