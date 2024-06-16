// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/pipeline"
	"github.com/harness/gitness/app/api/controller/trigger"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type pipelineRequest struct {
	repoRequest
	Identifier string `path:"pipeline_identifier"`
}

type executionRequest struct {
	pipelineRequest
	Number string `path:"execution_number"`
}

type triggerRequest struct {
	pipelineRequest
	Identifier string `path:"trigger_identifier"`
}

type logRequest struct {
	executionRequest
	StageNum string `path:"stage_number"`
	StepNum  string `path:"step_number"`
}

type createExecutionRequest struct {
	pipelineRequest
}

type createTriggerRequest struct {
	pipelineRequest
	trigger.CreateInput
}

type createPipelineRequest struct {
	repoRequest
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

type updateTriggerRequest struct {
	triggerRequest
	trigger.UpdateInput
}

type updatePipelineRequest struct {
	pipelineRequest
	pipeline.UpdateInput
}

var queryParameterLatest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamLatest,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Whether to fetch latest build information for each pipeline."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeBoolean),
			},
		},
	},
}

var queryParameterBranch = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamBranch,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Branch to run the execution for."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
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
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/pipelines", opCreate)

	opPipelines := openapi3.Operation{}
	opPipelines.WithTags("pipeline")
	opPipelines.WithMapOfAnything(map[string]interface{}{"operationId": "listPipelines"})
	opPipelines.WithParameters(queryParameterQueryRepo, QueryParameterPage, QueryParameterLimit, queryParameterLatest)
	_ = reflector.SetRequest(&opPipelines, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opPipelines, []types.Pipeline{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pipelines", opPipelines)

	opFind := openapi3.Operation{}
	opFind.WithTags("pipeline")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findPipeline"})
	_ = reflector.SetRequest(&opFind, new(getPipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Pipeline), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pipelines/{pipeline_identifier}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("pipeline")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deletePipeline"})
	_ = reflector.SetRequest(&opDelete, new(getPipelineRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/pipelines/{pipeline_identifier}", opDelete)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}", opUpdate)

	executionCreate := openapi3.Operation{}
	executionCreate.WithTags("pipeline")
	executionCreate.WithParameters(queryParameterBranch)
	executionCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createExecution"})
	_ = reflector.SetRequest(&executionCreate, new(createExecutionRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&executionCreate, new(types.Execution), http.StatusCreated)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions", executionCreate)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions/{execution_number}", executionFind)

	executionCancel := openapi3.Operation{}
	executionCancel.WithTags("pipeline")
	executionCancel.WithMapOfAnything(map[string]interface{}{"operationId": "cancelExecution"})
	_ = reflector.SetRequest(&executionCancel, new(getExecutionRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&executionCancel, new(types.Execution), http.StatusOK)
	_ = reflector.SetJSONResponse(&executionCancel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionCancel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionCancel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionCancel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions/{execution_number}/cancel", executionCancel)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions/{execution_number}", executionDelete)

	executionList := openapi3.Operation{}
	executionList.WithTags("pipeline")
	executionList.WithMapOfAnything(map[string]interface{}{"operationId": "listExecutions"})
	executionList.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&executionList, new(pipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&executionList, []types.Execution{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&executionList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions", executionList)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/triggers", triggerCreate)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/triggers/{trigger_identifier}", triggerFind)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/triggers/{trigger_identifier}", triggerDelete)

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
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/triggers/{trigger_identifier}", triggerUpdate)

	triggerList := openapi3.Operation{}
	triggerList.WithTags("pipeline")
	triggerList.WithMapOfAnything(map[string]interface{}{"operationId": "listTriggers"})
	triggerList.WithParameters(queryParameterQueryRepo, QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&triggerList, new(pipelineRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&triggerList, []types.Trigger{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&triggerList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/triggers", triggerList)

	logView := openapi3.Operation{}
	logView.WithTags("pipeline")
	logView.WithMapOfAnything(map[string]interface{}{"operationId": "viewLogs"})
	_ = reflector.SetRequest(&logView, new(logRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&logView, http.StatusOK, "application/json")
	_ = reflector.SetJSONResponse(&logView, []*livelog.Line{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&logView, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodGet,
		"/repos/{repo_ref}/pipelines/{pipeline_identifier}/executions/{execution_number}/logs/{stage_number}/{step_number}",
		logView,
	)
}
