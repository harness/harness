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

	"github.com/harness/gitness/app/api/usererror"

	"github.com/swaggest/openapi-go/openapi3"
)

func resourceOperations(reflector *openapi3.Reflector) {
	opListGitignore := openapi3.Operation{}
	opListGitignore.WithTags("resource")
	opListGitignore.WithMapOfAnything(map[string]any{"operationId": "listGitignore"})
	_ = reflector.SetRequest(&opListGitignore, new(gitignoreRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListGitignore, []string{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/resources/gitignore", opListGitignore)

	opListLicenses := openapi3.Operation{}
	opListLicenses.WithTags("resource")
	opListLicenses.WithMapOfAnything(map[string]any{"operationId": "listLicenses"})
	_ = reflector.SetRequest(&opListLicenses, new(licenseRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLicenses, []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLicenses, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLicenses, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLicenses, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/resources/license", opListLicenses)
}
