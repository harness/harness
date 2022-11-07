// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"
)

const (
	PathParamSpaceRef = "spaceRef"
)

func GetSpaceRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamSpaceRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	return url.PathUnescape(rawRef)
}
