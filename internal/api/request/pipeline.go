// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"
	"strconv"
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

// TODO: Move into separate execution folder
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

// TODO: Add list filters
// // ParseSortRepo extracts the repo sort parameter from the url.
// func ParseSortRepo(r *http.Request) enum.RepoAttr {
// 	return enum.ParseRepoAtrr(
// 		r.URL.Query().Get(QueryParamSort),
// 	)
// }

// // ParseRepoFilter extracts the repository filter from the url.
// func ParseRepoFilter(r *http.Request) *types.RepoFilter {
// 	return &types.RepoFilter{
// 		Query: ParseQuery(r),
// 		Order: ParseOrder(r),
// 		Page:  ParsePage(r),
// 		Sort:  ParseSortRepo(r),
// 		Size:  ParseLimit(r),
// 	}
// }
