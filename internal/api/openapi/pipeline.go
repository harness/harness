// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/controller/trigger"
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

type triggerRequest struct {
	pipelineRequest
	Ref string `path:"trigger_ref"`
}

type logRequest struct {
	executionRequest
	StageNum string `path:"stage_number"`
	StepNum  string `path:"step_number"`
}

type createExecutionRequest struct {
	pipelineRequest
	execution.CreateInput
}

type createTriggerRequest struct {
	pipelineRequest
	trigger.CreateInput
}

type createPipelineRequest struct {
	pipeline.CreateInput
}

type getExecutionRequest struct {
	executionRequest
}

type getTriggerRequest struct {
	triggerRequest
}

type getPipelineRequest struct {
	pipelineRequest
}

type updateExecutionRequest struct {
	executionRequest
	execution.UpdateInput
}

type updateTriggerRequest struct {
	triggerRequest
	trigger.UpdateInput
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

	triggerCreate := openapi3.Operation{}
	triggerCreate.WithTags("pipeline")
	triggerCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createTrigger"})
	_ = reflector.SetRequest(&triggerCreate, new(createTriggerRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&triggerCreate, new(types.Trigger), http.StatusCreated)
	_ = reflector.SetJSONResponse(&triggerCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&triggerCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/pipelines/{pipeline_ref}/triggers", triggerCreate)

	triggerFind := openapi3.Operation{}
	triggerFind.WithTags("pipeline")
	triggerFind.WithMapOfAnything(map[string]interface{}{"operationId": "findTrigger"})
	_ = reflector.SetRequest(&triggerFind, new(getTriggerRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&triggerFind, new(types.Trigger), http.StatusOK)
	_ = reflector.SetJSONResponse(&triggerFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&triggerFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/pipelines/{pipeline_ref}/triggers/{trigger_ref}", triggerFind)

	triggerDelete := openapi3.Operation{}
	triggerDelete.WithTags("pipeline")
	triggerDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteTrigger"})
	_ = reflector.SetRequest(&triggerDelete, new(getTriggerRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&triggerDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&triggerDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&triggerDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/pipelines/{pipeline_ref}/triggers/{trigger_ref}", triggerDelete)

	triggerUpdate := openapi3.Operation{}
	triggerUpdate.WithTags("pipeline")
	triggerUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateTrigger"})
	_ = reflector.SetRequest(&triggerUpdate, new(updateTriggerRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(types.Trigger), http.StatusOK)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&triggerUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/pipelines/{pipeline_ref}/triggers/{trigger_ref}", triggerUpdate)

	triggerList := openapi3.Operation{}
	triggerList.WithTags("pipeline")
	triggerList.WithMapOfAnything(map[string]interface{}{"operationId": "listTriggers"})
	triggerList.WithParameters(queryParameterQueryRepo, queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&triggerList, new(pipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&triggerList, []types.Trigger{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/pipelines/{pipeline_ref}/triggers", triggerList)

	logView := openapi3.Operation{}
	logView.WithTags("pipeline")
	logView.WithMapOfAnything(map[string]interface{}{"operationId": "viewLogs"})
	_ = reflector.SetRequest(&logView, new(logRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&logView, http.StatusOK, "text/plain")
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/pipelines/{pipeline_ref}/executions/{execution_number}/logs/{stage_number}/{step_number}", logView)
}
