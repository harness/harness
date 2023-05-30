// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/check"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type reportStatusCheckResultRequest struct {
	repoRequest
	CommitSHA string `path:"commit_sha"`
	check.ReportInput
}

type listStatusCheckResultsRequest struct {
	repoRequest
	CommitSHA string `path:"commit_sha"`
}

func checkOperations(reflector *openapi3.Reflector) {
	const tag = "status_checks"

	reportStatusCheckResults := openapi3.Operation{}
	reportStatusCheckResults.WithTags(tag)
	reportStatusCheckResults.WithMapOfAnything(map[string]interface{}{"operationId": "reportStatusCheckResults"})
	_ = reflector.SetRequest(&reportStatusCheckResults, new(reportStatusCheckResultRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(types.Check), http.StatusOK)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut, "/repos/{repo_ref}/checks/commits/{commit_sha}",
		reportStatusCheckResults)

	listStatusCheckResults := openapi3.Operation{}
	listStatusCheckResults.WithTags(tag)
	listStatusCheckResults.WithMapOfAnything(map[string]interface{}{"operationId": "listStatusCheckResults"})
	_ = reflector.SetRequest(&listStatusCheckResults, new(listStatusCheckResultsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new([]types.Check), http.StatusOK)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/checks/commits/{commit_sha}",
		listStatusCheckResults)
}
