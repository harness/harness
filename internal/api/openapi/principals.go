// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type (
	// principalRequest is the request for finding a principal.
	principalRequest struct {
		PrincipalUID string `path:"principal_uid"`
	}
)

// buildPrincipals function that constructs the openapi specification
// for principal resources.
func buildPrincipals(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("principals")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getPrincipal"})
	_ = reflector.SetRequest(&opFind, new(principalRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.PrincipalInfo), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/principals/{principal_uid}", opFind)
}
