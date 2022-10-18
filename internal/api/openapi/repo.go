// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/swaggest/openapi-go/openapi3"
)

type createRepositoryRequest struct {
	repo.CreateInput
}

type gitignoreRequest struct {
}

type licenseRequest struct {
}

type repoRequest struct {
	Ref string `json:"ref" path:"ref"`
}

type updateRepoRequest struct {
	repoRequest
	repo.UpdateInput
}

type moveRepoRequest struct {
	repoRequest
	repo.MoveInput
}

type createRepoPathRequest struct {
	repoRequest
	repo.CreatePathInput
}

type deleteRepoPathRequest struct {
	repoRequest
	PathID string `json:"pathID" path:"pathID"`
}

//nolint:funlen
func repoOperations(reflector *openapi3.Reflector) {
	createRepository := openapi3.Operation{}
	createRepository.WithTags("repository")
	createRepository.WithMapOfAnything(map[string]interface{}{"operationId": "createRepository"})
	_ = reflector.SetRequest(&createRepository, new(createRepositoryRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createRepository, new(types.Repository), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos", createRepository)

	opFind := openapi3.Operation{}
	opFind.WithTags("repository")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findRepository"})
	_ = reflector.SetRequest(&opFind, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{ref}", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("repository")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateRepository"})
	_ = reflector.SetRequest(&opUpdate, new(updateRepoRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{ref}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("repository")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteRepository"})
	_ = reflector.SetRequest(&opDelete, new(repoRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{ref}", opDelete)

	opMove := openapi3.Operation{}
	opMove.WithTags("repository")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveRepository"})
	_ = reflector.SetRequest(&opMove, new(moveRepoRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{ref}/move", opMove)

	opServiceAccounts := openapi3.Operation{}
	opServiceAccounts.WithTags("repository")
	opServiceAccounts.WithMapOfAnything(map[string]interface{}{"operationId": "listRepositoryServiceAccounts"})
	_ = reflector.SetRequest(&opServiceAccounts, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opServiceAccounts, []types.ServiceAccount{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{ref}/serviceAccounts", opServiceAccounts)

	opListPaths := openapi3.Operation{}
	opListPaths.WithTags("repository")
	opListPaths.WithMapOfAnything(map[string]interface{}{"operationId": "listRepositoryPaths"})
	_ = reflector.SetRequest(&opListPaths, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListPaths, []types.Path{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{ref}/paths", opListPaths)

	opCreatePath := openapi3.Operation{}
	opCreatePath.WithTags("repository")
	opCreatePath.WithMapOfAnything(map[string]interface{}{"operationId": "createRepositoryPath"})
	_ = reflector.SetRequest(&opCreatePath, new(createRepoPathRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreatePath, new(types.Path), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{ref}/paths", opCreatePath)

	onDeletePath := openapi3.Operation{}
	onDeletePath.WithTags("repository")
	onDeletePath.WithMapOfAnything(map[string]interface{}{"operationId": "deleteRepositoryPath"})
	_ = reflector.SetRequest(&onDeletePath, new(deleteRepoPathRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&onDeletePath, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{ref}/paths/{pathID}", onDeletePath)
}
