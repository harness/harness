// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/harness/gitness/types"
)

const (
	PipelinePathRef = "pipeline_ref"
	PipelineUID     = "pipeline_uid"
	ExecutionNumber = "execution_number"
)

func GetPipelinePathRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PipelinePathRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

func GetExecutionNumberFromPath(r *http.Request) (int64, error) {
	rawRef, err := PathParamOrError(r, ExecutionNumber)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(rawRef)
	if err != nil {
		return 0, err
	}

	// paths are unescaped
	return int64(n), nil
}

func GetPipelineUIDFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PipelineUID)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

// ParsePipelineFilter extracts the pipeline filter from the url.
func ParsePipelineFilter(r *http.Request) *types.PipelineFilter {
	return &types.PipelineFilter{
		Query: ParseQuery(r),
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
	}
}

// ParseExecutionFilter extracts the execution filter from the url.
func ParseExecutionFilter(r *http.Request) *types.ExecutionFilter {
	return &types.ExecutionFilter{
		Page: ParsePage(r),
		Size: ParseLimit(r),
	}
}
