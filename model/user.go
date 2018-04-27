// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"errors"
	"regexp"
)

// validate a username (e.g. from github)
var reUsername = regexp.MustCompile("^[a-zA-Z0-9-_.]+$")

var errUserLoginInvalid = errors.New("Invalid User Login")

// User represents a registered user.
//
// swagger:model user
type User struct {
	// the id for this user.
	//
	// required: true
	ID int64 `json:"id" meddler:"user_id,pk"`

	// Login is the username for this user.
	//
	// required: true
	Login string `json:"login"  meddler:"user_login"`

	// Token is the oauth2 token.
	Token string `json:"-"  meddler:"user_token"`

	// Secret is the oauth2 token secret.
	Secret string `json:"-" meddler:"user_secret"`

	// Expiry is the token and secret expiration timestamp.
	Expiry int64 `json:"-" meddler:"user_expiry"`

	// Email is the email address for this user.
	//
	// required: true
	Email string `json:"email" meddler:"user_email"`

	// the avatar url for this user.
	Avatar string `json:"avatar_url" meddler:"user_avatar"`

	// Activate indicates the user is active in the system.
	Active bool `json:"active" meddler:"user_active"`

	// Synced is the timestamp when the user was synced with the remote system.
	Synced int64 `json:"synced" meddler:"user_synced"`

	// Admin indicates the user is a system administrator.
	//
	// NOTE: This is sourced from the DRONE_ADMINS environment variable and is no
	// longer persisted in the database.
	Admin bool `json:"admin,omitempty" meddler:"-"`

	// Hash is a unique token used to sign tokens.
	Hash string `json:"-" meddler:"user_hash"`

	// DEPRECATED Admin indicates the user is a system administrator.
	XAdmin bool `json:"-" meddler:"user_admin"`
}

// Validate validates the required fields and formats.
func (u *User) Validate() error {
	switch {
	case len(u.Login) == 0:
		return errUserLoginInvalid
	case len(u.Login) > 250:
		return errUserLoginInvalid
	case !reUsername.MatchString(u.Login):
		return errUserLoginInvalid
	default:
		return nil
	}
}
