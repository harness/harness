// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/pkg/api/controller/check"
	"github.com/harness/gitness/pkg/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

func checkOperations(reflector *openapi3.Reflector) {
	const tag = "status_checks"

	reportStatusCheckResults := openapi3.Operation{}
	reportStatusCheckResults.WithTags(tag)
	reportStatusCheckResults.WithMapOfAnything(map[string]interface{}{"operationId": "reportStatusCheckResults"})
	_ = reflector.SetRequest(&reportStatusCheckResults, struct {
		repoRequest
		CommitSHA string `path:"commit_sha"`
		check.ReportInput
	}{}, http.MethodPut)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(types.Check), http.StatusOK)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reportStatusCheckResults, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut, "/repos/{repo_ref}/checks/commits/{commit_sha}",
		reportStatusCheckResults)

	listStatusCheckResults := openapi3.Operation{}
	listStatusCheckResults.WithTags(tag)
	listStatusCheckResults.WithParameters(
		queryParameterPage, queryParameterLimit)
	listStatusCheckResults.WithMapOfAnything(map[string]interface{}{"operationId": "listStatusCheckResults"})
	_ = reflector.SetRequest(&listStatusCheckResults, struct {
		repoRequest
		CommitSHA string `path:"commit_sha"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new([]types.Check), http.StatusOK)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listStatusCheckResults, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/checks/commits/{commit_sha}",
		listStatusCheckResults)
}
