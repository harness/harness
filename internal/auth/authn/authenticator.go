// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"net/http"

	"github.com/harness/gitness/types"
)

type Authenticator interface {
	Authenticate(r *http.Request) (*types.User, error)
}
