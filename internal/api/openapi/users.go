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

type (
	// request for finding or deleting a user.
	userRequest struct {
		Param string `path:"email"`
	}

	// request for listing users.
	userListRequest struct {
		Sort  string `query:"sort"      enum:"id,email,created,updated"`
		Order string `query:"order"     enum:"asc,desc"`

		// include pagination request
		paginationRequest
	}
)

// helper function that constructs the openapi specification
// for user resources.
func buildUsers(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("users")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getUserEmail"})
	_ = reflector.SetRequest(&opFind, new(userRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/users/{email}", opFind)

	opList := openapi3.Operation{}
	opList.WithTags("users")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listUsers"})
	_ = reflector.SetRequest(&opList, new(userListRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opList, new([]*types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/users", opList)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("users")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createUser"})
	_ = reflector.SetRequest(&opCreate, new(types.UserInput), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/users", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("users")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateUsers"})
	_ = reflector.SetRequest(&opUpdate, new(user.UpdateInput), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/users/{email}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("users")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteUser"})
	_ = reflector.SetRequest(&opDelete, new(userRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/users/{email}", opDelete)
}
