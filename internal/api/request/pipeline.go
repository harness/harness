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
	PipelinePathRef = "pipeline_ref"
	PipelineUID     = "pipeline_uid"
	ExecutionNumber = "execution_number"
)

func GetPipelineRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PipelinePathRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

func GetExecutionNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, ExecutionNumber)
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
