// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type pipelineRequest struct {
	Ref string `path:"pipeline_ref"`
}

type executionRequest struct {
	pipelineRequest
	Number string `path:"execution_number"`
}

type createExecutionRequest struct {
	pipelineRequest
	execution.CreateInput
}

type createPipelineRequest struct {
	pipeline.CreateInput
}

type getExecutionRequest struct {
	executionRequest
}

type getPipelineRequest struct {
	pipelineRequest
}

type updateExecutionRequest struct {
	executionRequest
	execution.UpdateInput
}

type updatePipelineRequest struct {
	pipelineRequest
	pipeline.UpdateInput
}

func pipelineOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("pipeline")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createPipeline"})
	_ = reflector.SetRequest(&opCreate, new(createPipelineRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.Pipeline), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/pipelines", opCreate)

	opFind := openapi3.Operation{}
	opFind.WithTags("pipeline")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findPipeline"})
	_ = reflector.SetRequest(&opFind, new(getPipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Pipeline), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/pipelines/{pipeline_ref}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("pipeline")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deletePipeline"})
	_ = reflector.SetRequest(&opDelete, new(getPipelineRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/pipelines/{pipeline_ref}", opDelete)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("pipeline")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updatePipeline"})
	_ = reflector.SetRequest(&opUpdate, new(updatePipelineRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Pipeline), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/pipelines/{pipeline_ref}", opUpdate)

	executionCreate := openapi3.Operation{}
	executionCreate.WithTags("pipeline")
	executionCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createExecution"})
	_ = reflector.SetRequest(&executionCreate, new(createExecutionRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&executionCreate, new(types.Execution), http.StatusCreated)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/pipelines/{pipeline_ref}/executions", executionCreate)

	executionFind := openapi3.Operation{}
	executionFind.WithTags("pipeline")
	executionFind.WithMapOfAnything(map[string]interface{}{"operationId": "findExecution"})
	_ = reflector.SetRequest(&executionFind, new(getExecutionRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&executionFind, new(types.Execution), http.StatusOK)
	_ = reflector.SetJSONResponse(&executionFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/pipelines/{pipeline_ref}/executions/{execution_number}", executionFind)

	executionDelete := openapi3.Operation{}
	executionDelete.WithTags("pipeline")
	executionDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteExecution"})
	_ = reflector.SetRequest(&executionDelete, new(getExecutionRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&executionDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&executionDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/pipelines/{pipeline_ref}/executions/{execution_number}", executionDelete)

	executionUpdate := openapi3.Operation{}
	executionUpdate.WithTags("pipeline")
	executionUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateExecution"})
	_ = reflector.SetRequest(&executionUpdate, new(updateExecutionRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&executionUpdate, new(types.Execution), http.StatusOK)
	_ = reflector.SetJSONResponse(&executionUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&executionUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/pipelines/{pipeline_ref}/executions/{execution_number}", executionUpdate)

	executionList := openapi3.Operation{}
	executionList.WithTags("pipeline")
	executionList.WithMapOfAnything(map[string]interface{}{"operationId": "listExecutions"})
	executionList.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&executionList, new(pipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&executionList, []types.Execution{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/pipelines/{pipeline_ref}/executions", executionList)
}
