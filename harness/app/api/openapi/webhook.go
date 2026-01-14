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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

// webhookType is used to add has_secret field.
type webhookType struct {
	types.Webhook
	HasSecret bool `json:"has_secret"`
}

type createSpaceWebhookRequest struct {
	spaceRequest
	types.WebhookCreateInput
}

type createRepoWebhookRequest struct {
	repoRequest
	types.WebhookCreateInput
}

type listSpaceWebhooksRequest struct {
	spaceRequest
}

type listRepoWebhooksRequest struct {
	repoRequest
}

type spaceWebhookRequest struct {
	spaceRequest
	ID int64 `path:"webhook_identifier"`
}

type repoWebhookRequest struct {
	repoRequest
	ID int64 `path:"webhook_identifier"`
}

type getSpaceWebhookRequest struct {
	spaceWebhookRequest
}

type getRepoWebhookRequest struct {
	repoWebhookRequest
}

type deleteSpaceWebhookRequest struct {
	spaceWebhookRequest
}

type deleteRepoWebhookRequest struct {
	repoWebhookRequest
}

type updateSpaceWebhookRequest struct {
	spaceWebhookRequest
	types.WebhookUpdateInput
}

type updateRepoWebhookRequest struct {
	repoWebhookRequest
	types.WebhookUpdateInput
}

type listSpaceWebhookExecutionsRequest struct {
	spaceWebhookRequest
}

type listRepoWebhookExecutionsRequest struct {
	repoWebhookRequest
}

type spaceWebhookExecutionRequest struct {
	spaceWebhookRequest
	ID int64 `path:"webhook_execution_id"`
}

type repoWebhookExecutionRequest struct {
	repoWebhookRequest
	ID int64 `path:"webhook_execution_id"`
}

type getSpaceWebhookExecutionRequest struct {
	spaceWebhookExecutionRequest
}

type getRepoWebhookExecutionRequest struct {
	repoWebhookExecutionRequest
}

var queryParameterSortWebhook = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the webhooks are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.WebhookAttrIdentifier.String()),
				Enum: []interface{}{
					// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
					ptr.String(enum.WebhookAttrID.String()),
					ptr.String(enum.WebhookAttrUID.String()),
					// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
					ptr.String(enum.WebhookAttrDisplayName.String()),
					ptr.String(enum.WebhookAttrCreated.String()),
					ptr.String(enum.WebhookAttrUpdated.String()),
				},
			},
		},
	},
}

var queryParameterQueryWebhook = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the webhooks by their identifier."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

//nolint:funlen
func webhookOperations(reflector *openapi3.Reflector) {
	// space

	createSpaceWebhook := openapi3.Operation{}
	createSpaceWebhook.WithTags("webhook")
	createSpaceWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "createSpaceWebhook"})
	_ = reflector.SetRequest(&createSpaceWebhook, new(createSpaceWebhookRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createSpaceWebhook, new(webhookType), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createSpaceWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createSpaceWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createSpaceWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createSpaceWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/webhooks", createSpaceWebhook)

	listSpaceWebhooks := openapi3.Operation{}
	listSpaceWebhooks.WithTags("webhook")
	listSpaceWebhooks.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaceWebhooks"})
	listSpaceWebhooks.WithParameters(queryParameterQueryWebhook, queryParameterSortWebhook, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listSpaceWebhooks, new(listSpaceWebhooksRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listSpaceWebhooks, new([]webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&listSpaceWebhooks, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listSpaceWebhooks, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listSpaceWebhooks, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listSpaceWebhooks, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/webhooks", listSpaceWebhooks)

	getSpaceWebhook := openapi3.Operation{}
	getSpaceWebhook.WithTags("webhook")
	getSpaceWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "getSpaceWebhook"})
	_ = reflector.SetRequest(&getSpaceWebhook, new(getSpaceWebhookRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getSpaceWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&getSpaceWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getSpaceWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getSpaceWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getSpaceWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/webhooks/{webhook_identifier}", getSpaceWebhook)

	updateSpaceWebhook := openapi3.Operation{}
	updateSpaceWebhook.WithTags("webhook")
	updateSpaceWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "updateWebhook"})
	_ = reflector.SetRequest(&updateSpaceWebhook, new(updateSpaceWebhookRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateSpaceWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&updateSpaceWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&updateSpaceWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&updateSpaceWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&updateSpaceWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(
		http.MethodPatch, "/spaces/{space_ref}/webhooks/{webhook_identifier}", updateSpaceWebhook,
	)

	deleteSpaceWebhook := openapi3.Operation{}
	deleteSpaceWebhook.WithTags("webhook")
	deleteSpaceWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "deleteWebhook"})
	_ = reflector.SetRequest(&deleteSpaceWebhook, new(deleteSpaceWebhookRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&deleteSpaceWebhook, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&deleteSpaceWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&deleteSpaceWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&deleteSpaceWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&deleteSpaceWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/spaces/{space_ref}/webhooks/{webhook_identifier}", deleteSpaceWebhook,
	)

	listSpaceWebhookExecutions := openapi3.Operation{}
	listSpaceWebhookExecutions.WithTags("webhook")
	listSpaceWebhookExecutions.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaceWebhookExecutions"})
	listSpaceWebhookExecutions.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listSpaceWebhookExecutions, new(listSpaceWebhookExecutionsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listSpaceWebhookExecutions, new([]types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&listSpaceWebhookExecutions, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listSpaceWebhookExecutions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listSpaceWebhookExecutions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listSpaceWebhookExecutions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/spaces/{space_ref}/webhooks/{webhook_identifier}/executions", listSpaceWebhookExecutions)

	getSpaceWebhookExecution := openapi3.Operation{}
	getSpaceWebhookExecution.WithTags("webhook")
	getSpaceWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "getSpaceWebhookExecution"})
	getSpaceWebhookExecution.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&getSpaceWebhookExecution, new(getSpaceWebhookExecutionRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getSpaceWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&getSpaceWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getSpaceWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getSpaceWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getSpaceWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/spaces/{space_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}",
		getSpaceWebhookExecution,
	)

	retriggerSpaceWebhookExecution := openapi3.Operation{}
	retriggerSpaceWebhookExecution.WithTags("webhook")
	retriggerSpaceWebhookExecution.WithMapOfAnything(
		map[string]interface{}{"operationId": "retriggerSpaceWebhookExecution"},
	)
	_ = reflector.SetRequest(&retriggerSpaceWebhookExecution, new(spaceWebhookExecutionRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&retriggerSpaceWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&retriggerSpaceWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&retriggerSpaceWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&retriggerSpaceWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&retriggerSpaceWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/spaces/{space_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}/retrigger",
		retriggerSpaceWebhookExecution,
	)

	// repo

	createRepoWebhook := openapi3.Operation{}
	createRepoWebhook.WithTags("webhook")
	createRepoWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "createRepoWebhook"})
	_ = reflector.SetRequest(&createRepoWebhook, new(createRepoWebhookRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createRepoWebhook, new(webhookType), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createRepoWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createRepoWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createRepoWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createRepoWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/webhooks", createRepoWebhook)

	listRepoWebhooks := openapi3.Operation{}
	listRepoWebhooks.WithTags("webhook")
	listRepoWebhooks.WithMapOfAnything(map[string]interface{}{"operationId": "listRepoWebhooks"})
	listRepoWebhooks.WithParameters(queryParameterQueryWebhook, queryParameterSortWebhook, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listRepoWebhooks, new(listRepoWebhooksRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listRepoWebhooks, new([]webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&listRepoWebhooks, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listRepoWebhooks, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listRepoWebhooks, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listRepoWebhooks, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/webhooks", listRepoWebhooks)

	getRepoWebhook := openapi3.Operation{}
	getRepoWebhook.WithTags("webhook")
	getRepoWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "getRepoWebhook"})
	_ = reflector.SetRequest(&getRepoWebhook, new(getRepoWebhookRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getRepoWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&getRepoWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getRepoWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getRepoWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getRepoWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/webhooks/{webhook_identifier}", getRepoWebhook)

	updateRepoWebhook := openapi3.Operation{}
	updateRepoWebhook.WithTags("webhook")
	updateRepoWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "updateRepoWebhook"})
	_ = reflector.SetRequest(&updateRepoWebhook, new(updateRepoWebhookRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateRepoWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&updateRepoWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&updateRepoWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&updateRepoWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&updateRepoWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/webhooks/{webhook_identifier}", updateRepoWebhook)

	deleteRepoWebhook := openapi3.Operation{}
	deleteRepoWebhook.WithTags("webhook")
	deleteRepoWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "deleteRepoWebhook"})
	_ = reflector.SetRequest(&deleteRepoWebhook, new(deleteRepoWebhookRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&deleteRepoWebhook, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&deleteRepoWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&deleteRepoWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&deleteRepoWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&deleteRepoWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/repos/{repo_ref}/webhooks/{webhook_identifier}", deleteRepoWebhook,
	)

	listRepoWebhookExecutions := openapi3.Operation{}
	listRepoWebhookExecutions.WithTags("webhook")
	listRepoWebhookExecutions.WithMapOfAnything(map[string]interface{}{"operationId": "listRepoWebhookExecutions"})
	listRepoWebhookExecutions.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listRepoWebhookExecutions, new(listRepoWebhookExecutionsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listRepoWebhookExecutions, new([]types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&listRepoWebhookExecutions, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listRepoWebhookExecutions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listRepoWebhookExecutions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listRepoWebhookExecutions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions", listRepoWebhookExecutions)

	getRepoWebhookExecution := openapi3.Operation{}
	getRepoWebhookExecution.WithTags("webhook")
	getRepoWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "getRepoWebhookExecution"})
	getRepoWebhookExecution.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&getRepoWebhookExecution, new(getRepoWebhookExecutionRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getRepoWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&getRepoWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getRepoWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getRepoWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getRepoWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}", getRepoWebhookExecution)

	retriggerRepoWebhookExecution := openapi3.Operation{}
	retriggerRepoWebhookExecution.WithTags("webhook")
	retriggerRepoWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "retriggerRepoWebhookExecution"})
	_ = reflector.SetRequest(&retriggerRepoWebhookExecution, new(repoWebhookExecutionRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&retriggerRepoWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&retriggerRepoWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&retriggerRepoWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&retriggerRepoWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&retriggerRepoWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}/retrigger",
		retriggerRepoWebhookExecution)
}
