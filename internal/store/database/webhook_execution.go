// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.WebhookExecutionStore = (*WebhookExecutionStore)(nil)

// NewWebhookExecutionStore returns a new WebhookExecutionStore.
func NewWebhookExecutionStore(db *sqlx.DB) *WebhookExecutionStore {
	return &WebhookExecutionStore{
		db: db,
	}
}

// WebhookExecutionStore implements store.WebhookExecution backed by a relational database.
type WebhookExecutionStore struct {
	db *sqlx.DB
}

// webhookExecution is used to store executions of webhooks
// The object should be later re-packed into a different struct to return it as an API response.
type webhookExecution struct {
	ID                 int64                       `db:"webhook_execution_id"`
	RetriggerOf        null.Int                    `db:"webhook_execution_retrigger_of"`
	Retriggerable      bool                        `db:"webhook_execution_retriggerable"`
	WebhookID          int64                       `db:"webhook_execution_webhook_id"`
	TriggerType        enum.WebhookTrigger         `db:"webhook_execution_trigger_type"`
	TriggerID          string                      `db:"webhook_execution_trigger_id"`
	Result             enum.WebhookExecutionResult `db:"webhook_execution_result"`
	Created            int64                       `db:"webhook_execution_created"`
	Duration           int64                       `db:"webhook_execution_duration"`
	Error              string                      `db:"webhook_execution_error"`
	RequestURL         string                      `db:"webhook_execution_request_url"`
	RequestHeaders     string                      `db:"webhook_execution_request_headers"`
	RequestBody        string                      `db:"webhook_execution_request_body"`
	ResponseStatusCode int64                       `db:"webhook_execution_response_status_code"`
	ResponseStatus     string                      `db:"webhook_execution_response_status"`
	ResponseHeaders    string                      `db:"webhook_execution_response_headers"`
	ResponseBody       string                      `db:"webhook_execution_response_body"`
}

const (
	webhookExecutionColumns = `
		 webhook_execution_id
		,webhook_execution_retrigger_of
		,webhook_execution_retriggerable
		,webhook_execution_webhook_id
		,webhook_execution_trigger_type
		,webhook_execution_trigger_id
		,webhook_execution_result
		,webhook_execution_created
		,webhook_execution_duration
		,webhook_execution_error
		,webhook_execution_request_url
		,webhook_execution_request_headers
		,webhook_execution_request_body
		,webhook_execution_response_status_code
		,webhook_execution_response_status
		,webhook_execution_response_headers
		,webhook_execution_response_body`

	webhookExecutionSelectBase = `
	SELECT` + webhookExecutionColumns + `
	FROM webhook_executions`
)

// Find finds the webhook execution by id.
func (s *WebhookExecutionStore) Find(ctx context.Context, id int64) (*types.WebhookExecution, error) {
	const sqlQuery = webhookExecutionSelectBase + `
	WHERE webhook_execution_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &webhookExecution{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return mapToWebhookExecution(dst), nil
}

// Create creates a new webhook execution entry.
func (s *WebhookExecutionStore) Create(ctx context.Context, execution *types.WebhookExecution) error {
	const sqlQuery = `
	INSERT INTO webhook_executions (
		 webhook_execution_retrigger_of
		,webhook_execution_retriggerable
		,webhook_execution_webhook_id
		,webhook_execution_trigger_type
		,webhook_execution_trigger_id
		,webhook_execution_result
		,webhook_execution_created
		,webhook_execution_duration
		,webhook_execution_error
		,webhook_execution_request_url
		,webhook_execution_request_headers
		,webhook_execution_request_body
		,webhook_execution_response_status_code
		,webhook_execution_response_status
		,webhook_execution_response_headers
		,webhook_execution_response_body
	) values (
		 :webhook_execution_retrigger_of
		,:webhook_execution_retriggerable
		,:webhook_execution_webhook_id
		,:webhook_execution_trigger_type
		,:webhook_execution_trigger_id
		,:webhook_execution_result
		,:webhook_execution_created
		,:webhook_execution_duration
		,:webhook_execution_error
		,:webhook_execution_request_url
		,:webhook_execution_request_headers
		,:webhook_execution_request_body
		,:webhook_execution_response_status_code
		,:webhook_execution_response_status
		,:webhook_execution_response_headers
		,:webhook_execution_response_body
	) RETURNING webhook_execution_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapToInternalWebhookExecution(execution))
	if err != nil {
		return processSQLErrorf(err, "Failed to bind webhook execution object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&execution.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// ListForWebhook lists the webhook executions for a given webhook id.
func (s *WebhookExecutionStore) ListForWebhook(ctx context.Context, webhookID int64,
	opts *types.WebhookExecutionFilter) ([]*types.WebhookExecution, error) {
	stmt := builder.
		Select(webhookExecutionColumns).
		From("webhook_executions").
		Where("webhook_execution_webhook_id = ?", webhookID)

	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	// fixed ordering by desc id (new ones first) - add customized ordering if deemed necessary.
	stmt = stmt.OrderBy("webhook_execution_id DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*webhookExecution{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return mapToWebhookExecutions(dst), nil
}

// ListForTrigger lists the webhook executions for a given trigger id.
func (s *WebhookExecutionStore) ListForTrigger(ctx context.Context,
	triggerID string) ([]*types.WebhookExecution, error) {
	const sqlQuery = webhookExecutionSelectBase + `
	WHERE webhook_execution_trigger_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*webhookExecution{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, triggerID); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return mapToWebhookExecutions(dst), nil
}

func mapToWebhookExecution(execution *webhookExecution) *types.WebhookExecution {
	return &types.WebhookExecution{
		ID:            execution.ID,
		RetriggerOf:   execution.RetriggerOf.Ptr(),
		Retriggerable: execution.Retriggerable,
		Created:       execution.Created,
		WebhookID:     execution.WebhookID,
		TriggerType:   execution.TriggerType,
		TriggerID:     execution.TriggerID,
		Result:        execution.Result,
		Error:         execution.Error,
		Duration:      execution.Duration,
		Request: types.WebhookExecutionRequest{
			URL:     execution.RequestURL,
			Headers: execution.RequestHeaders,
			Body:    execution.RequestBody,
		},
		Response: types.WebhookExecutionResponse{
			StatusCode: execution.ResponseStatusCode,
			Status:     execution.ResponseStatus,
			Headers:    execution.ResponseHeaders,
			Body:       execution.ResponseBody,
		},
	}
}

func mapToInternalWebhookExecution(execution *types.WebhookExecution) *webhookExecution {
	return &webhookExecution{
		ID:                 execution.ID,
		RetriggerOf:        null.IntFromPtr(execution.RetriggerOf),
		Retriggerable:      execution.Retriggerable,
		Created:            execution.Created,
		WebhookID:          execution.WebhookID,
		TriggerType:        execution.TriggerType,
		TriggerID:          execution.TriggerID,
		Result:             execution.Result,
		Error:              execution.Error,
		Duration:           execution.Duration,
		RequestURL:         execution.Request.URL,
		RequestHeaders:     execution.Request.Headers,
		RequestBody:        execution.Request.Body,
		ResponseStatusCode: execution.Response.StatusCode,
		ResponseStatus:     execution.Response.Status,
		ResponseHeaders:    execution.Response.Headers,
		ResponseBody:       execution.Response.Body,
	}
}

func mapToWebhookExecutions(executions []*webhookExecution) []*types.WebhookExecution {
	m := make([]*types.WebhookExecution, len(executions))
	for i, hook := range executions {
		m[i] = mapToWebhookExecution(hook)
	}

	return m
}
