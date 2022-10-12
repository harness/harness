// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
)

const (
	PathParamUserUID           = "userUID"
	PathParamServiceAccountUID = "saUID"
)

func GetUserUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamUserUID)
}

func GetServiceAccountUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamServiceAccountUID)
}
