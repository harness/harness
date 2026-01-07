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

package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/golang-jwt/jwt/v5"
)

const (
	//TODO: Update when ready to change repo and build
	issuer = "Gitness"
)

// Source represents the source of the SubClaimsAccessPermissions.
type Source string

const (
	OciSource Source = "oci"
)

// Claims defines Harness jwt claims.
type Claims struct {
	jwt.RegisteredClaims

	PrincipalID int64 `json:"pid,omitempty"`

	Token             *SubClaimsToken             `json:"tkn,omitempty"`
	Membership        *SubClaimsMembership        `json:"ms,omitempty"`
	AccessPermissions *SubClaimsAccessPermissions `json:"ap,omitempty"`
}

// SubClaimsToken contains information about the token the JWT was created for.
type SubClaimsToken struct {
	Type enum.TokenType `json:"typ,omitempty"`
	ID   int64          `json:"id,omitempty"`
}

// SubClaimsMembership contains the ephemeral membership the JWT was created with.
type SubClaimsMembership struct {
	Role    enum.MembershipRole `json:"role,omitempty"`
	SpaceID int64               `json:"sid,omitempty"`
}

// SubClaimsAccessPermissions stores allowed actions on a resource.
type SubClaimsAccessPermissions struct {
	Source      Source              `json:"src,omitempty"`
	Permissions []AccessPermissions `json:"permissions,omitempty"`
}

// AccessPermissions stores allowed actions on a resource.
type AccessPermissions struct {
	SpaceID     int64             `json:"sid,omitempty"`
	Permissions []enum.Permission `json:"p"`
}

// extractFirstSecretFromList extracts the first secret from a comma-separated string.
// This is a helper function to support JWT secret rotation.
func extractFirstSecretFromList(secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("empty secret provided")
	}

	// If no comma in the string, just trim and return directly.
	if !strings.Contains(secret, ",") {
		trimmed := strings.TrimSpace(secret)
		if trimmed == "" {
			return "", fmt.Errorf("secret cannot be empty")
		}
		return trimmed, nil
	}

	parts := strings.Split(secret, ",")
	firstSecret := strings.TrimSpace(parts[0])

	if firstSecret == "" {
		return "", fmt.Errorf("first secret in list cannot be empty")
	}

	return firstSecret, nil
}

// GenerateForToken generates a jwt for a given token.
func GenerateForToken(token *types.Token, secret string) (string, error) {
	// Use the first secret for signing (support for rotation)
	signingSecret, err := extractFirstSecretFromList(secret)
	if err != nil {
		return "", fmt.Errorf("failed to get first secret: %w", err)
	}

	var expiresAt *jwt.NumericDate
	if token.ExpiresAt != nil {
		expiresAt = jwt.NewNumericDate(time.UnixMilli(*token.ExpiresAt))
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer,
			// times required to be in sec not millisec
			IssuedAt:  jwt.NewNumericDate(time.UnixMilli(token.IssuedAt)),
			ExpiresAt: expiresAt,
		},
		PrincipalID: token.PrincipalID,
		Token: &SubClaimsToken{
			Type: token.Type,
			ID:   token.ID,
		},
	})

	res, err := jwtToken.SignedString([]byte(signingSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return res, nil
}

// GenerateWithMembership generates a jwt with the given ephemeral membership.
func GenerateWithMembership(
	principalID int64,
	spaceID int64,
	role enum.MembershipRole,
	lifetime time.Duration,
	secret string,
) (string, error) {
	// Use the first secret for signing (support for rotation)
	signingSecret, err := extractFirstSecretFromList(secret)
	if err != nil {
		return "", fmt.Errorf("failed to get first secret: %w", err)
	}

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(lifetime)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer,
			// times required to be in sec
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		PrincipalID: principalID,
		Membership: &SubClaimsMembership{
			SpaceID: spaceID,
			Role:    role,
		},
	})

	res, err := jwtToken.SignedString([]byte(signingSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return res, nil
}

// GenerateForTokenWithAccessPermissions generates a jwt for a given token.
func GenerateForTokenWithAccessPermissions(
	principalID int64,
	lifetime *time.Duration,
	secret string, accessPermissions *SubClaimsAccessPermissions,
) (string, error) {
	// Use the first secret for signing (support for rotation)
	signingSecret, err := extractFirstSecretFromList(secret)
	if err != nil {
		return "", fmt.Errorf("failed to get first secret: %w", err)
	}

	issuedAt := time.Now()
	if lifetime == nil {
		return "", fmt.Errorf("token lifetime is required")
	}
	expiresAt := issuedAt.Add(*lifetime)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		PrincipalID:       principalID,
		AccessPermissions: accessPermissions,
	})

	res, err := jwtToken.SignedString([]byte(signingSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return res, nil
}
