// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// Represents the type of the JWT token.
type TokenType string

const (
	// TokenTypeSession is the token returned during user login or signup.
	TokenTypeSession TokenType = "session"

	// TokenTypePAT is a personal access token.
	TokenTypePAT TokenType = "pat"

	// TokenTypeSAT is a service account access token.
	TokenTypeSAT TokenType = "sat"

	// TokenTypeOAuth2 is the token returned to an oauth client.
	TokenTypeOAuth2 TokenType = "oauth2"
)
