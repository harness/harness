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

	"github.com/harness/gitness/app/api/controller/webhook"
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

type createWebhookRequest struct {
	repoRequest
	webhook.CreateInput
}

type listWebhooksRequest struct {
	repoRequest
}

type webhookRequest struct {
	repoRequest
	ID int64 `path:"webhook_identifier"`
}

type getWebhookRequest struct {
	webhookRequest
}

type deleteWebhookRequest struct {
	webhookRequest
}

type updateWebhookRequest struct {
	webhookRequest
	webhook.UpdateInput
}

type listWebhookExecutionsRequest struct {
	webhookRequest
}

type webhookExecutionRequest struct {
	webhookRequest
	ID int64 `path:"webhook_execution_id"`
}

type getWebhookExecutionRequest struct {
	webhookExecutionRequest
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
	createWebhook := openapi3.Operation{}
	createWebhook.WithTags("webhook")
	createWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "createWebhook"})
	_ = reflector.SetRequest(&createWebhook, new(createWebhookRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createWebhook, new(webhookType), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/webhooks", createWebhook)

	listWebhooks := openapi3.Operation{}
	listWebhooks.WithTags("webhook")
	listWebhooks.WithMapOfAnything(map[string]interface{}{"operationId": "listWebhooks"})
	listWebhooks.WithParameters(queryParameterQueryWebhook, queryParameterSortWebhook, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listWebhooks, new(listWebhooksRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listWebhooks, new([]webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&listWebhooks, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listWebhooks, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listWebhooks, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listWebhooks, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/webhooks", listWebhooks)

	getWebhook := openapi3.Operation{}
	getWebhook.WithTags("webhook")
	getWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "getWebhook"})
	_ = reflector.SetRequest(&getWebhook, new(getWebhookRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&getWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/webhooks/{webhook_identifier}", getWebhook)

	updateWebhook := openapi3.Operation{}
	updateWebhook.WithTags("webhook")
	updateWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "updateWebhook"})
	_ = reflector.SetRequest(&updateWebhook, new(updateWebhookRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/webhooks/{webhook_identifier}", updateWebhook)

	deleteWebhook := openapi3.Operation{}
	deleteWebhook.WithTags("webhook")
	deleteWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "deleteWebhook"})
	_ = reflector.SetRequest(&deleteWebhook, new(deleteWebhookRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&deleteWebhook, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/webhooks/{webhook_identifier}", deleteWebhook)

	listWebhookExecutions := openapi3.Operation{}
	listWebhookExecutions.WithTags("webhook")
	listWebhookExecutions.WithMapOfAnything(map[string]interface{}{"operationId": "listWebhookExecutions"})
	listWebhookExecutions.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listWebhookExecutions, new(listWebhookExecutionsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new([]types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions", listWebhookExecutions)

	getWebhookExecution := openapi3.Operation{}
	getWebhookExecution.WithTags("webhook")
	getWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "getWebhookExecution"})
	getWebhookExecution.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&getWebhookExecution, new(getWebhookExecutionRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}", getWebhookExecution)

	retriggerWebhookExecution := openapi3.Operation{}
	retriggerWebhookExecution.WithTags("webhook")
	retriggerWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "retriggerWebhookExecution"})
	_ = reflector.SetRequest(&retriggerWebhookExecution, nil, http.MethodPost)
	_ = reflector.SetJSONResponse(&retriggerWebhookExecution, new(types.WebhookExecution), http.StatusOK)
	_ = reflector.SetJSONResponse(&retriggerWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&retriggerWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&retriggerWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&retriggerWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_identifier}/executions/{webhook_execution_id}", retriggerWebhookExecution)
}
