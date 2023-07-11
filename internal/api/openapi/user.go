// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/swaggest/openapi-go/openapi3"
)

type createTokenRequest struct {
	user.CreateTokenInput
}

// helper function that constructs the openapi specification
// for user account resources.
func buildUser(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("user")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getUser"})
	_ = reflector.SetRequest(&opFind, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/user", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("user")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateUser"})
	_ = reflector.SetRequest(&opUpdate, new(user.UpdateInput), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/user", opUpdate)

	opToken := openapi3.Operation{}
	opToken.WithTags("user")
	opToken.WithMapOfAnything(map[string]interface{}{"operationId": "createToken"})
	_ = reflector.SetRequest(&opToken, new(createTokenRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opToken, new(types.TokenResponse), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opToken, new(usererror.Error), http.StatusInternalServerError)
}
