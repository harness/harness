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
	// adminUsersCreateRequest is the request for the admin user create operation.
	adminUsersCreateRequest struct {
		user.CreateInput
	}

	// adminUsersRequest is the request for user specific admin user operations.
	adminUsersRequest struct {
		UserUID string `path:"user_uid"`
	}

	// adminUsersUpdateRequest is the request for the admin user update operation.
	adminUsersUpdateRequest struct {
		adminUsersRequest
		user.UpdateInput
	}

	// adminUserListRequest is the request for listing users.
	adminUserListRequest struct {
		Sort  string `query:"sort"      enum:"id,email,created,updated"`
		Order string `query:"order"     enum:"asc,desc"`

		// include pagination request
		paginationRequest
	}
)

// helper function that constructs the openapi specification
// for admin resources.
func buildAdmin(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("admin")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "adminGetUser"})
	_ = reflector.SetRequest(&opFind, new(adminUsersRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/admin/users/{user_uid}", opFind)

	opList := openapi3.Operation{}
	opList.WithTags("admin")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "adminListUsers"})
	_ = reflector.SetRequest(&opList, new(adminUserListRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opList, new([]*types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/admin/users", opList)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("admin")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "adminCreateUser"})
	_ = reflector.SetRequest(&opCreate, new(adminUsersCreateRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/admin/users", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("admin")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "adminUpdateUser"})
	_ = reflector.SetRequest(&opUpdate, new(adminUsersUpdateRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/admin/users/{user_uid}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("admin")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "adminDeleteUser"})
	_ = reflector.SetRequest(&opDelete, new(adminUsersRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/admin/users/{user_uid}", opDelete)
}
