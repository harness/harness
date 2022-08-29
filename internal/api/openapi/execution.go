// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type (
	// request to find or delete a execution.
	executionRequest struct {
		Pipeline  string `path:"pipeline"`
		Execution string `path:"execution"`

		// include account parameters
		baseRequest
	}

	// request to list all executions
	executionListRequest struct {
		Pipeline string `path:"pipeline"`

		// include pagination
		paginationRequest

		// include account parameters
		baseRequest
	}

	// request to create a execution.
	executionCreateRequest struct {
		Pipeline string `path:"pipeline"`

		// include request body input.
		types.ExecutionInput

		// include account parameters
		baseRequest
	}

	// request to update a execution.
	executionUpdateRequest struct {
		Pipeline  string `path:"pipeline"`
		Execution string `path:"execution"`

		// include request body input.
		types.ExecutionInput

		// include account parameters
		baseRequest
	}
)

// helper function that constructs the openapi specification
// for execution resources.
func buildExecution(reflector *openapi3.Reflector) {

	opFind := openapi3.Operation{}
	opFind.WithTags("execution")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getExecution"})
	reflector.SetRequest(&opFind, new(executionRequest), http.MethodGet)
	reflector.SetJSONResponse(&opFind, new(types.Execution), http.StatusOK)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodGet, "/pipelines/{pipeline}/executions/{execution}", opFind)

	opList := openapi3.Operation{}
	opList.WithTags("execution")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listExecutions"})
	reflector.SetRequest(&opList, new(executionListRequest), http.MethodGet)
	reflector.SetJSONResponse(&opList, new([]*types.Execution), http.StatusOK)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodGet, "/pipelines/{pipeline}/executions", opList)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("execution")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createExecution"})
	reflector.SetRequest(&opCreate, new(executionCreateRequest), http.MethodPost)
	reflector.SetJSONResponse(&opCreate, new(types.Execution), http.StatusOK)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPost, "/pipelines/{pipeline}/executions", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("execution")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateExecution"})
	reflector.SetRequest(&opUpdate, new(executionUpdateRequest), http.MethodPatch)
	reflector.SetJSONResponse(&opUpdate, new(types.Execution), http.StatusOK)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPatch, "/pipelines/{pipeline}/executions/{execution}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("execution")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteExecution"})
	reflector.SetRequest(&opDelete, new(executionRequest), http.MethodDelete)
	reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodDelete, "/pipelines/{pipeline}/executions/{execution}", opDelete)
}
