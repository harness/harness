// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"

	"github.com/harness/gitness/types"
)

const (
	SecretRef = "secret_ref"
)

func GetSecretRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, SecretRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

// ParseExecutionFilter extracts the execution filter from the url.
func ParseSecretFilter(r *http.Request) *types.SecretFilter {
	return &types.SecretFilter{
		Page: ParsePage(r),
		Size: ParseLimit(r),
	}
}
