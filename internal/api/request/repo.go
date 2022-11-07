// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"
)

const (
	PathParamRepoRef = "repositoryRef"
)

func GetRepoRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamRepoRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}
