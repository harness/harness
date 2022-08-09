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

type (
	// request for finding or deleting a user.
	userRequest struct {
		Param string `path:"email"`
	}

	// request for listing users.
	userListRequest struct {
		Sort  string `query:"sort"      enum:"id,email,created,updated"`
		Order string `query:"direction" enum:"asc,desc"`

		// include pagination request
		paginationRequest
	}

	// request for updating a user.
	userUpdateRequest struct {
		Param string `path:"email"`

		// include request body input.
		types.UserInput
	}
)

// helper function that constructs the openapi specification
// for user resources.
func buildUsers(reflector *openapi3.Reflector) {

	opFind := openapi3.Operation{}
	opFind.WithTags("users")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getUserEmail"})
	reflector.SetRequest(&opFind, new(userRequest), http.MethodGet)
	reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodGet, "/users/{email}", opFind)

	opList := openapi3.Operation{}
	opList.WithTags("users")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listUsers"})
	reflector.SetRequest(&opList, new(userListRequest), http.MethodGet)
	reflector.SetJSONResponse(&opList, new([]*types.User), http.StatusOK)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodGet, "/users", opList)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("users")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createUser"})
	reflector.SetRequest(&opCreate, new(types.UserInput), http.MethodPost)
	reflector.SetJSONResponse(&opCreate, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPost, "/users", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("users")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateUsers"})
	reflector.SetRequest(&opUpdate, new(userUpdateRequest), http.MethodPatch)
	reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPatch, "/users/{email}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("users")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteUser"})
	reflector.SetRequest(&opDelete, new(userRequest), http.MethodDelete)
	reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodDelete, "/users/{email}", opDelete)
}
