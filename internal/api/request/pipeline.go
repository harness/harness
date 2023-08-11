// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"
)

const (
	PathParamPipelineRef     = "pipeline_ref"
	PathParamExecutionNumber = "execution_number"
)

func GetPipelineRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamPipelineRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

func GetExecutionNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamExecutionNumber)
}
