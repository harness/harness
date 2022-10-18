// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"
)

//nolint:funlen
func resourceOperations(reflector *openapi3.Reflector) {
	opListGitignore := openapi3.Operation{}
	opListGitignore.WithTags("resource")
	opListGitignore.WithMapOfAnything(map[string]interface{}{"operationId": "listGitignore"})
	_ = reflector.SetRequest(&opListGitignore, new(gitignoreRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListGitignore, []string{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListGitignore, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/resources/gitignore", opListGitignore)

	opListLicenses := openapi3.Operation{}
	opListLicenses.WithTags("resource")
	opListLicenses.WithMapOfAnything(map[string]interface{}{"operationId": "listLicenses"})
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
