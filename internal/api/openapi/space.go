// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/swaggest/openapi-go/openapi3"
)

type createSpaceRequest struct {
	space.CreateInput
}

type spaceRequest struct {
	Ref string `json:"ref" path:"spaceRef"`
}

type updateSpaceRequest struct {
	spaceRequest
	space.UpdateInput
}

type moveSpaceRequest struct {
	spaceRequest
	space.MoveInput
}

type createPathRequest struct {
	spaceRequest
	space.CreatePathInput
}

type deletePathRequest struct {
	spaceRequest
	PathID string `json:"pathID" path:"pathID"`
}

//nolint:funlen // api spec generation no need for checking func complexity
func spaceOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("space")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createSpace"})
	_ = reflector.SetRequest(&opCreate, new(createSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.Space), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces", opCreate)

	opGet := openapi3.Operation{}
	opGet.WithTags("space")
	opGet.WithMapOfAnything(map[string]interface{}{"operationId": "getSpace"})
	_ = reflector.SetRequest(&opGet, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGet, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{spaceRef}", opGet)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("space")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateSpace"})
	_ = reflector.SetRequest(&opUpdate, new(updateSpaceRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/spaces/{spaceRef}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("space")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteSpace"})
	_ = reflector.SetRequest(&opDelete, new(spaceRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{spaceRef}", opDelete)

	opMove := openapi3.Operation{}
	opMove.WithTags("space")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveSpace"})
	_ = reflector.SetRequest(&opMove, new(moveSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{spaceRef}/move", opMove)

	opSpaces := openapi3.Operation{}
	opSpaces.WithTags("space")
	opSpaces.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaces"})
	opSpaces.WithParameters(queryParameterPage, queryParameterPerPage)
	_ = reflector.SetRequest(&opSpaces, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSpaces, []types.Space{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{spaceRef}/spaces", opSpaces)

	opRepos := openapi3.Operation{}
	opRepos.WithTags("space")
	opRepos.WithMapOfAnything(map[string]interface{}{"operationId": "listRepos"})
	opRepos.WithParameters(queryParameterPage, queryParameterPerPage)
	_ = reflector.SetRequest(&opRepos, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opRepos, []types.Repository{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{spaceRef}/repos", opRepos)

	opServiceAccounts := openapi3.Operation{}
	opServiceAccounts.WithTags("space")
	opServiceAccounts.WithMapOfAnything(map[string]interface{}{"operationId": "listServiceAccounts"})
	_ = reflector.SetRequest(&opServiceAccounts, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opServiceAccounts, []types.ServiceAccount{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{spaceRef}/serviceAccounts", opServiceAccounts)

	opListPaths := openapi3.Operation{}
	opListPaths.WithTags("space")
	opListPaths.WithMapOfAnything(map[string]interface{}{"operationId": "listPaths"})
	_ = reflector.SetRequest(&opListPaths, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListPaths, []types.Path{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{spaceRef}/paths", opListPaths)

	opCreatePath := openapi3.Operation{}
	opCreatePath.WithTags("space")
	opCreatePath.WithMapOfAnything(map[string]interface{}{"operationId": "createPath"})
	_ = reflector.SetRequest(&opCreatePath, new(createPathRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreatePath, new(types.Path), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{spaceRef}/paths", opCreatePath)

	onDeletePath := openapi3.Operation{}
	onDeletePath.WithTags("space")
	onDeletePath.WithMapOfAnything(map[string]interface{}{"operationId": "deletePath"})
	_ = reflector.SetRequest(&onDeletePath, new(deletePathRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&onDeletePath, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{spaceRef}/paths/{pathID}", onDeletePath)
}
