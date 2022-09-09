// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"net/http"

	"github.com/harness/gitness/types"
)

/*
 * An abstraction of an entity thats responsible for authenticating users
 * that are making calls via HTTP.
 */
type Authenticator interface {
	/*
	 * Tries to authenticate a user if credentials are available.
	 * Returns:
	 *		(user, nil) - request contained auth data and user was verified
	 *		(nil, err)  - request contained auth data but verification failed
	 *		(nil, nil)	- request didn't contain any auth data
	 */
	Authenticate(r *http.Request) (*types.User, error)
}
