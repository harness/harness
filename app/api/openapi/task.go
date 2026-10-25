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

	"github.com/harness/gitness/app/api/controller/task"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type taskRequest struct {
	repoRequest
	Number int64 `path:"task_number"`
}

type createTaskRequest struct {
	repoRequest
	task.CreateInput
}

type listTasksRequest struct {
	repoRequest
}

type listSpaceTasksRequest struct {
	spaceRequest
}

type updateTaskRequest struct {
	taskRequest
	task.UpdateInput
}

type stateTaskRequest struct {
	taskRequest
	task.StateInput
}

type resumeTaskRequest struct {
	taskRequest
	task.ResumeInput
}

type commentTaskRequest struct {
	taskRequest
	task.CommentInput
}

func taskListQueryParameter(name, description string, schema *openapi3.Schema) openapi3.ParameterOrRef {
	return openapi3.ParameterOrRef{
		Parameter: &openapi3.Parameter{
			Name:        name,
			In:          openapi3.ParameterInQuery,
			Description: ptr.String(description),
			Required:    ptr.Bool(false),
			Schema:      &openapi3.SchemaOrRef{Schema: schema},
			Style:       ptr.String(string(openapi3.EncodingStyleForm)),
			Explode:     ptr.Bool(true),
		},
	}
}

func taskArrayOf(item *openapi3.Schema) *openapi3.Schema {
	return &openapi3.Schema{
		Type:  ptrSchemaType(openapi3.SchemaTypeArray),
		Items: &openapi3.SchemaOrRef{Schema: item},
	}
}

var (
	queryParameterTaskState = taskListQueryParameter(request.QueryParamState,
		"The states of the tasks to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.TaskState("").Enum(),
		}))
	queryParameterTaskPriority = taskListQueryParameter(request.QueryParamTaskPriority,
		"The priorities of the tasks to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.TaskPriority("").Enum(),
		}))
	queryParameterTaskOrigin = taskListQueryParameter(request.QueryParamTaskOrigin,
		"The origins of the tasks to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.TaskOrigin("").Enum(),
		}))
	queryParameterTaskAssigneeID = taskListQueryParameter(request.QueryParamTaskAssigneeID,
		"The ids of the principals the tasks are assigned to.",
		taskArrayOf(&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeInteger)}))
	queryParameterTaskAssigneeGroupID = taskListQueryParameter(request.QueryParamTaskAssigneeGroupID,
		"The id of the user group the tasks are assigned to.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeInteger)})
	queryParameterTaskIsAgent = taskListQueryParameter(request.QueryParamTaskIsAgent,
		"Only tasks assigned (true) or not assigned (false) to the agent.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeBoolean)})
	queryParameterTaskLabelID = taskListQueryParameter(request.QueryParamLabelID,
		"The ids of the labels used to filter tasks.",
		taskArrayOf(&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeInteger)}))
	queryParameterTaskQuery = taskListQueryParameter(request.QueryParamQuery,
		"The substring by which the task titles are filtered.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeString)})
	queryParameterTaskRecursive = taskListQueryParameter(request.QueryParamRecursive,
		"Include the tasks of the repositories of the descendant spaces.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeBoolean), Default: ptrptr(false)})
)

func setTaskErrorResponses(reflector *openapi3.Reflector, op *openapi3.Operation, withNotFound bool) {
	_ = reflector.SetJSONResponse(op, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(op, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(op, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(op, new(usererror.Error), http.StatusForbidden)
	if withNotFound {
		_ = reflector.SetJSONResponse(op, new(usererror.Error), http.StatusNotFound)
	}
}

//nolint:funlen
func taskOperations(reflector *openapi3.Reflector) {
	taskFilterParameters := []openapi3.ParameterOrRef{
		queryParameterTaskState, queryParameterTaskPriority, queryParameterTaskOrigin,
		queryParameterTaskAssigneeID, queryParameterTaskAssigneeGroupID, queryParameterTaskIsAgent,
		queryParameterTaskLabelID, queryParameterTaskQuery,
		QueryParameterPage, QueryParameterLimit,
	}

	listTasks := openapi3.Operation{}
	listTasks.WithTags("task")
	listTasks.WithMapOfAnything(map[string]any{"operationId": "listTasks"})
	listTasks.WithParameters(taskFilterParameters...)
	_ = reflector.SetRequest(&listTasks, new(listTasksRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listTasks, new([]types.Task), http.StatusOK)
	setTaskErrorResponses(reflector, &listTasks, false)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tasks", listTasks)

	listSpaceTasks := openapi3.Operation{}
	listSpaceTasks.WithTags("space")
	listSpaceTasks.WithMapOfAnything(map[string]any{"operationId": "listSpaceTasks"})
	listSpaceTasks.WithParameters(append(taskFilterParameters, queryParameterTaskRecursive)...)
	_ = reflector.SetRequest(&listSpaceTasks, new(listSpaceTasksRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listSpaceTasks, new([]types.TaskRepo), http.StatusOK)
	setTaskErrorResponses(reflector, &listSpaceTasks, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/tasks", listSpaceTasks)

	createTask := openapi3.Operation{}
	createTask.WithTags("task")
	createTask.WithMapOfAnything(map[string]any{"operationId": "createTask"})
	_ = reflector.SetRequest(&createTask, new(createTaskRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createTask, new(types.Task), http.StatusCreated)
	setTaskErrorResponses(reflector, &createTask, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/tasks", createTask)

	getTask := openapi3.Operation{}
	getTask.WithTags("task")
	getTask.WithMapOfAnything(map[string]any{"operationId": "getTask"})
	_ = reflector.SetRequest(&getTask, new(taskRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getTask, new(types.Task), http.StatusOK)
	setTaskErrorResponses(reflector, &getTask, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tasks/{task_number}", getTask)

	updateTask := openapi3.Operation{}
	updateTask.WithTags("task")
	updateTask.WithMapOfAnything(map[string]any{"operationId": "updateTask"})
	_ = reflector.SetRequest(&updateTask, new(updateTaskRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateTask, new(types.Task), http.StatusOK)
	setTaskErrorResponses(reflector, &updateTask, true)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/tasks/{task_number}", updateTask)

	stateTask := openapi3.Operation{}
	stateTask.WithTags("task")
	stateTask.WithMapOfAnything(map[string]any{"operationId": "stateTask"})
	_ = reflector.SetRequest(&stateTask, new(stateTaskRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&stateTask, new(types.Task), http.StatusOK)
	setTaskErrorResponses(reflector, &stateTask, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/tasks/{task_number}/state", stateTask)

	resumeTask := openapi3.Operation{}
	resumeTask.WithTags("task")
	resumeTask.WithMapOfAnything(map[string]any{"operationId": "resumeTask"})
	_ = reflector.SetRequest(&resumeTask, new(resumeTaskRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&resumeTask, new(types.Task), http.StatusOK)
	setTaskErrorResponses(reflector, &resumeTask, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/tasks/{task_number}/resume", resumeTask)

	listTaskActivities := openapi3.Operation{}
	listTaskActivities.WithTags("task")
	listTaskActivities.WithMapOfAnything(map[string]any{"operationId": "listTaskActivities"})
	listTaskActivities.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listTaskActivities, new(taskRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listTaskActivities, new([]types.TaskActivity), http.StatusOK)
	setTaskErrorResponses(reflector, &listTaskActivities, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tasks/{task_number}/activities", listTaskActivities)

	commentTask := openapi3.Operation{}
	commentTask.WithTags("task")
	commentTask.WithMapOfAnything(map[string]any{"operationId": "commentTask"})
	_ = reflector.SetRequest(&commentTask, new(commentTaskRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&commentTask, nil, http.StatusNoContent)
	setTaskErrorResponses(reflector, &commentTask, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/tasks/{task_number}/comments", commentTask)

	listTaskLinks := openapi3.Operation{}
	listTaskLinks.WithTags("task")
	listTaskLinks.WithMapOfAnything(map[string]any{"operationId": "listTaskLinks"})
	_ = reflector.SetRequest(&listTaskLinks, new(taskRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listTaskLinks, new([]types.TaskLink), http.StatusOK)
	setTaskErrorResponses(reflector, &listTaskLinks, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tasks/{task_number}/links", listTaskLinks)
}
