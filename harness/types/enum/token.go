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

package enum

// TokenType represents the type of the JWT token.
type TokenType string

const (
	// TokenTypeSession is the token returned during user login or signup.
	TokenTypeSession TokenType = "session"

	// TokenTypePAT is a personal access token.
	TokenTypePAT TokenType = "pat"

	// TokenTypeSAT is a service account access token.
	TokenTypeSAT TokenType = "sat"

	// TokenTypeRemoteAuth is the token returned during ssh git-lfs-authenticate.
	TokenTypeRemoteAuth TokenType = "remoteAuth"
)
