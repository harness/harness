// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/types"
)

var (
	// An error that is returned if the authorizer doesn't find any data in the request that can be used for auth.
	ErrNoAuthData = errors.New("The request doesn't contain any auth data that can be used by the Authorizer.")
)

/*
 * An abstraction of an entity thats responsible for authenticating users
 * that are making calls via HTTP.
 */
type Authenticator interface {
	/*
	 * Tries to authenticate a user if credentials are available.
	 * Returns:
	 *		(user, nil) 			- request contains auth data and user was verified
	 *		(nil, ErrNoAuthData)	- request doesn't contain any auth data
	 *		(nil, err)  			- request contains auth data but verification failed
	 */
	Authenticate(r *http.Request) (*types.User, error)
}
