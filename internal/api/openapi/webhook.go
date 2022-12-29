// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/swaggest/openapi-go/openapi3"
)

// webhookTrigger is a plugin for enum.WebhookTrigger to allow using oneof.
type webhookTrigger string

func (webhookTrigger) Enum() []interface{} {
	return toInterfaceSlice(enum.GetAllWebhookTriggers())
}

// webhookCreateInput is used to overshadow field Triggers of webhook.CreateInput.
type webhookCreateInput struct {
	webhook.CreateInput
	Triggers []webhookTrigger `json:"triggers"`
}

// webhookParent is a plugin for enum.WebhookParent to allow using oneof.
type webhookParent string

func (webhookParent) Enum() []interface{} {
	return toInterfaceSlice(enum.GetAllWebhookParents())
}

// webhookType is used to overshadow fields Parent & Triggers of types.Webhook.
type webhookType struct {
	types.Webhook
	ParentType webhookParent    `json:"parent_type"`
	Triggers   []webhookTrigger `json:"triggers"`
}

type createWebhookRequest struct {
	repoRequest
	webhookCreateInput
}

type listWebhooksRequest struct {
	repoRequest
}

type webhookRequest struct {
	repoRequest
	ID int64 `path:"webhook_id"`
}

type getWebhookRequest struct {
	webhookRequest
}

type deleteWebhookRequest struct {
	webhookRequest
}

// webhookUpdateInput is used to overshadow field Triggers of webhook.UpdateInput.
type webhookUpdateInput struct {
	webhook.UpdateInput
	Triggers []webhookTrigger `json:"triggers"`
}

type updateWebhookRequest struct {
	webhookRequest
	webhookUpdateInput
}

// webhookExecutionResult is a plugin for enum.WebhookExecutionResult to allow using oneof.
type webhookExecutionResult string

func (webhookExecutionResult) Enum() []interface{} {
	return toInterfaceSlice(enum.GetAllWebhookExecutionResults())
}

// webhookExecutionType is used to overshadow triggers TriggerType & Result of types.WebhookExecution.
type webhookExecutionType struct {
	types.WebhookExecution
	TriggerType webhookTrigger         `json:"trigger_type"`
	Result      webhookExecutionResult `json:"result"`
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

//nolint:funlen
func webhookOperations(reflector *openapi3.Reflector) {
	createWebhook := openapi3.Operation{}
	createWebhook.WithTags("webhook")
	createWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "createWebhook"})
	_ = reflector.SetRequest(&createWebhook, new(createWebhookRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/webhooks", createWebhook)

	listWebhooks := openapi3.Operation{}
	listWebhooks.WithTags("webhook")
	listWebhooks.WithMapOfAnything(map[string]interface{}{"operationId": "listWebhooks"})
	listWebhooks.WithParameters(queryParameterPage, queryParameterLimit)
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
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/webhooks/{webhook_id}", getWebhook)

	updateWebhook := openapi3.Operation{}
	updateWebhook.WithTags("webhook")
	updateWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "updateWebhook"})
	_ = reflector.SetRequest(&updateWebhook, new(updateWebhookRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateWebhook, new(webhookType), http.StatusOK)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&updateWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/webhooks/{webhook_id}", updateWebhook)

	deleteWebhook := openapi3.Operation{}
	deleteWebhook.WithTags("webhook")
	deleteWebhook.WithMapOfAnything(map[string]interface{}{"operationId": "deleteWebhook"})
	_ = reflector.SetRequest(&deleteWebhook, new(deleteWebhookRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&deleteWebhook, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&deleteWebhook, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/webhooks/{webhook_id}", deleteWebhook)

	listWebhookExecutions := openapi3.Operation{}
	listWebhookExecutions.WithTags("webhook")
	listWebhookExecutions.WithMapOfAnything(map[string]interface{}{"operationId": "listWebhookExecutions"})
	listWebhookExecutions.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&listWebhookExecutions, new(listWebhookExecutionsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new([]webhookExecutionType), http.StatusOK)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listWebhookExecutions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_id}/executions", listWebhookExecutions)

	getWebhookExecution := openapi3.Operation{}
	getWebhookExecution.WithTags("webhook")
	getWebhookExecution.WithMapOfAnything(map[string]interface{}{"operationId": "getWebhookExecution"})
	getWebhookExecution.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&getWebhookExecution, new(getWebhookExecutionRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(webhookExecutionType), http.StatusOK)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getWebhookExecution, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/webhooks/{webhook_id}/executions/{webhook_execution_id}", getWebhookExecution)
}
