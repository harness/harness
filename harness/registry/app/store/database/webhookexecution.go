//  Copyright 2023 Harness, Inc.
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

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

type WebhookExecutionDao struct {
	db *sqlx.DB
}

const (
	webhookExecutionColumns = `
		 registry_webhook_execution_id
		,registry_webhook_execution_retrigger_of
		,registry_webhook_execution_retriggerable
		,registry_webhook_execution_webhook_id
		,registry_webhook_execution_trigger_type
		,registry_webhook_execution_trigger_id
		,registry_webhook_execution_result
		,registry_webhook_execution_created
		,registry_webhook_execution_duration
		,registry_webhook_execution_error
		,registry_webhook_execution_request_url
		,registry_webhook_execution_request_headers
		,registry_webhook_execution_request_body
		,registry_webhook_execution_response_status_code
		,registry_webhook_execution_response_status
		,registry_webhook_execution_response_headers
		,registry_webhook_execution_response_body`

	webhookExecutionSelectBase = `
	SELECT` + webhookExecutionColumns + `
	FROM registry_webhook_executions`
)

func (w WebhookExecutionDao) Find(ctx context.Context, id int64) (*gitnesstypes.WebhookExecutionCore, error) {
	const sqlQuery = webhookExecutionSelectBase + `
	WHERE registry_webhook_execution_id = $1`

	db := dbtx.GetAccessor(ctx, w.db)

	dst := &webhookExecutionDB{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return mapToWebhookExecution(dst), nil
}

func (w WebhookExecutionDao) Create(ctx context.Context, webhookExecution *gitnesstypes.WebhookExecutionCore) error {
	const sqlQuery = `
		INSERT INTO registry_webhook_executions (
			registry_webhook_execution_retrigger_of
            ,registry_webhook_execution_retriggerable
            ,registry_webhook_execution_webhook_id
            ,registry_webhook_execution_trigger_type
            ,registry_webhook_execution_trigger_id
            ,registry_webhook_execution_result
            ,registry_webhook_execution_created
            ,registry_webhook_execution_duration
            ,registry_webhook_execution_error
            ,registry_webhook_execution_request_url
            ,registry_webhook_execution_request_headers
            ,registry_webhook_execution_request_body
            ,registry_webhook_execution_response_status_code
            ,registry_webhook_execution_response_status
            ,registry_webhook_execution_response_headers
            ,registry_webhook_execution_response_body
		) values (
			 :registry_webhook_execution_retrigger_of
            ,:registry_webhook_execution_retriggerable
            ,:registry_webhook_execution_webhook_id
            ,:registry_webhook_execution_trigger_type
            ,:registry_webhook_execution_trigger_id
            ,:registry_webhook_execution_result
            ,:registry_webhook_execution_created
            ,:registry_webhook_execution_duration
            ,:registry_webhook_execution_error
            ,:registry_webhook_execution_request_url
            ,:registry_webhook_execution_request_headers
            ,:registry_webhook_execution_request_body
            ,:registry_webhook_execution_response_status_code
            ,:registry_webhook_execution_response_status
            ,:registry_webhook_execution_response_headers
            ,:registry_webhook_execution_response_body
		) RETURNING registry_webhook_execution_id`

	db := dbtx.GetAccessor(ctx, w.db)

	dbwebhookExecution := mapToWebhookExecutionDB(webhookExecution)

	query, arg, err := db.BindNamed(sqlQuery, dbwebhookExecution)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to registry bind webhook object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&dbwebhookExecution.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

func (w WebhookExecutionDao) ListForWebhook(
	ctx context.Context,
	webhookID int64,
	limit int,
	page int,
	size int,
) ([]*gitnesstypes.WebhookExecutionCore, error) {
	stmt := database.Builder.
		Select(webhookExecutionColumns).
		From("registry_webhook_executions").
		Where("registry_webhook_execution_webhook_id = ?", webhookID)

	stmt = stmt.Limit(database.Limit(limit))
	stmt = stmt.Offset(database.Offset(page, size))

	// fixed ordering by desc id (new ones first) - add customized ordering if deemed necessary.
	stmt = stmt.OrderBy("registry_webhook_execution_id DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, w.db)

	dst := []*webhookExecutionDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return mapToWebhookExecutions(dst), nil
}

func (w WebhookExecutionDao) CountForWebhook(ctx context.Context, webhookID int64) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("registry_webhook_executions").
		Where("registry_webhook_execution_webhook_id = ?", webhookID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, w.db)

	var count int64
	if err = db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Count query failed")
	}

	return count, nil
}

func (w WebhookExecutionDao) ListForTrigger(
	ctx context.Context,
	triggerID string,
) ([]*gitnesstypes.WebhookExecutionCore, error) {
	const sqlQuery = webhookExecutionSelectBase + `
	WHERE registry_webhook_execution_trigger_id = $1`

	db := dbtx.GetAccessor(ctx, w.db)

	dst := []*webhookExecutionDB{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, triggerID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return mapToWebhookExecutions(dst), nil
}

func NewWebhookExecutionDao(db *sqlx.DB) store.WebhooksExecutionRepository {
	return &WebhookExecutionDao{
		db: db,
	}
}

type webhookExecutionDB struct {
	ID                 int64                       `db:"registry_webhook_execution_id"`
	RetriggerOf        null.Int                    `db:"registry_webhook_execution_retrigger_of"`
	Retriggerable      bool                        `db:"registry_webhook_execution_retriggerable"`
	WebhookID          int64                       `db:"registry_webhook_execution_webhook_id"`
	TriggerType        enum.WebhookTrigger         `db:"registry_webhook_execution_trigger_type"`
	TriggerID          string                      `db:"registry_webhook_execution_trigger_id"`
	Result             enum.WebhookExecutionResult `db:"registry_webhook_execution_result"`
	Created            int64                       `db:"registry_webhook_execution_created"`
	Duration           int64                       `db:"registry_webhook_execution_duration"`
	Error              string                      `db:"registry_webhook_execution_error"`
	RequestURL         string                      `db:"registry_webhook_execution_request_url"`
	RequestHeaders     string                      `db:"registry_webhook_execution_request_headers"`
	RequestBody        string                      `db:"registry_webhook_execution_request_body"`
	ResponseStatusCode int                         `db:"registry_webhook_execution_response_status_code"`
	ResponseStatus     string                      `db:"registry_webhook_execution_response_status"`
	ResponseHeaders    string                      `db:"registry_webhook_execution_response_headers"`
	ResponseBody       string                      `db:"registry_webhook_execution_response_body"`
}

func mapToWebhookExecution(webhookExecutionDB *webhookExecutionDB) *gitnesstypes.WebhookExecutionCore {
	webhookExecution := &gitnesstypes.WebhookExecutionCore{
		ID:            webhookExecutionDB.ID,
		RetriggerOf:   webhookExecutionDB.RetriggerOf.Ptr(),
		Retriggerable: webhookExecutionDB.Retriggerable,
		WebhookID:     webhookExecutionDB.WebhookID,
		TriggerType:   webhookExecutionDB.TriggerType,
		TriggerID:     webhookExecutionDB.TriggerID,
		Result:        webhookExecutionDB.Result,
		Created:       webhookExecutionDB.Created,
		Duration:      webhookExecutionDB.Duration,
		Error:         webhookExecutionDB.Error,
		Request: gitnesstypes.WebhookExecutionRequest{
			URL:     webhookExecutionDB.RequestURL,
			Headers: webhookExecutionDB.RequestHeaders,
			Body:    webhookExecutionDB.RequestBody,
		},
		Response: gitnesstypes.WebhookExecutionResponse{
			StatusCode: webhookExecutionDB.ResponseStatusCode,
			Status:     webhookExecutionDB.ResponseStatus,
			Headers:    webhookExecutionDB.ResponseHeaders,
			Body:       webhookExecutionDB.ResponseBody,
		},
	}
	return webhookExecution
}

func mapToWebhookExecutionDB(webhookExecution *gitnesstypes.WebhookExecutionCore) *webhookExecutionDB {
	webhookExecutionDD := &webhookExecutionDB{
		ID:                 webhookExecution.ID,
		RetriggerOf:        null.IntFromPtr(webhookExecution.RetriggerOf),
		Retriggerable:      webhookExecution.Retriggerable,
		WebhookID:          webhookExecution.WebhookID,
		TriggerType:        webhookExecution.TriggerType,
		TriggerID:          webhookExecution.TriggerID,
		Result:             webhookExecution.Result,
		Created:            webhookExecution.Created,
		Duration:           webhookExecution.Duration,
		Error:              webhookExecution.Error,
		RequestURL:         webhookExecution.Request.URL,
		RequestHeaders:     webhookExecution.Request.Headers,
		RequestBody:        webhookExecution.Request.Body,
		ResponseStatusCode: webhookExecution.Response.StatusCode,
		ResponseStatus:     webhookExecution.Response.Status,
		ResponseHeaders:    webhookExecution.Response.Headers,
		ResponseBody:       webhookExecution.Response.Body,
	}
	return webhookExecutionDD
}

func mapToWebhookExecutions(executions []*webhookExecutionDB) []*gitnesstypes.WebhookExecutionCore {
	m := make([]*gitnesstypes.WebhookExecutionCore, len(executions))
	for i, hook := range executions {
		m[i] = mapToWebhookExecution(hook)
	}
	return m
}
