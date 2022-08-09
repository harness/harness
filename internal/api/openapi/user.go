// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render"
	"github.com/bradrydzewski/my-app/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type currentUserResponse struct {
	Data   *types.User `json:"data"`
	Status string      `json:"status" enum:"SUCCESS,FAILURE,ERROR"`
}

// helper function that constructs the openapi specification
// for user account resources.
func buildUser(reflector *openapi3.Reflector) {

	opFind := openapi3.Operation{}
	opFind.WithTags("user")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getUser"})
	reflector.SetRequest(&opFind, nil, http.MethodGet)
	reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodGet, "/user", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("user")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateUser"})
	reflector.SetRequest(&opUpdate, new(types.UserInput), http.MethodPatch)
	reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodPatch, "/user", opUpdate)

	opToken := openapi3.Operation{}
	opToken.WithTags("user")
	opToken.WithMapOfAnything(map[string]interface{}{"operationId": "createToken"})
	reflector.SetRequest(&opToken, new(types.Token), http.MethodPost)
	reflector.SetJSONResponse(&opToken, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opToken, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodPost, "/user/token", opToken)

	opCurrent := openapi3.Operation{}
	opCurrent.WithTags("user")
	opCurrent.WithMapOfAnything(map[string]interface{}{"operationId": "getCurrentUser"})
	reflector.SetRequest(&opFind, new(baseRequest), http.MethodGet)
	reflector.SetJSONResponse(&opCurrent, new(currentUserResponse), http.StatusOK)
	reflector.SetJSONResponse(&opCurrent, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodGet, "/api/user/currentUser", opCurrent)
}
