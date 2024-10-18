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

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.WebhookStore = (*WebhookStore)(nil)

// NewWebhookStore returns a new WebhookStore.
func NewWebhookStore(db *sqlx.DB) *WebhookStore {
	return &WebhookStore{
		db: db,
	}
}

// WebhookStore implements store.Webhook backed by a relational database.
type WebhookStore struct {
	db *sqlx.DB
}

// webhook is an internal representation used to store webhook data in the database.
type webhook struct {
	ID        int64    `db:"webhook_id"`
	Version   int64    `db:"webhook_version"`
	RepoID    null.Int `db:"webhook_repo_id"`
	SpaceID   null.Int `db:"webhook_space_id"`
	CreatedBy int64    `db:"webhook_created_by"`
	Created   int64    `db:"webhook_created"`
	Updated   int64    `db:"webhook_updated"`
	Internal  bool     `db:"webhook_internal"`

	Identifier string `db:"webhook_uid"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName           string      `db:"webhook_display_name"`
	Description           string      `db:"webhook_description"`
	URL                   string      `db:"webhook_url"`
	Secret                string      `db:"webhook_secret"`
	Enabled               bool        `db:"webhook_enabled"`
	Insecure              bool        `db:"webhook_insecure"`
	Triggers              string      `db:"webhook_triggers"`
	LatestExecutionResult null.String `db:"webhook_latest_execution_result"`
}

const (
	webhookColumns = `
		 webhook_id
		,webhook_version
		,webhook_repo_id
		,webhook_space_id
		,webhook_created_by
		,webhook_created
		,webhook_updated
		,webhook_uid
		,webhook_display_name
		,webhook_description
		,webhook_url
		,webhook_secret
		,webhook_enabled
		,webhook_insecure
		,webhook_triggers
		,webhook_latest_execution_result
		,webhook_internal`

	webhookSelectBase = `
	SELECT` + webhookColumns + `
	FROM webhooks`
)

// Find finds the webhook by id.
func (s *WebhookStore) Find(ctx context.Context, id int64) (*types.Webhook, error) {
	const sqlQuery = webhookSelectBase + `
		WHERE webhook_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &webhook{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	res, err := mapToWebhook(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map webhook to external type: %w", err)
	}

	return res, nil
}

// FindByIdentifier finds the webhook with the given Identifier for the given parent.
func (s *WebhookStore) FindByIdentifier(
	ctx context.Context,
	parentType enum.WebhookParent,
	parentID int64,
	identifier string,
) (*types.Webhook, error) {
	stmt := database.Builder.
		Select(webhookColumns).
		From("webhooks").
		Where("LOWER(webhook_uid) = ?", strings.ToLower(identifier))

	switch parentType {
	case enum.WebhookParentRepo:
		stmt = stmt.Where("webhook_repo_id = ?", parentID)
	case enum.WebhookParentSpace:
		stmt = stmt.Where("webhook_space_id = ?", parentID)
	default:
		return nil, fmt.Errorf("webhook parent type '%s' is not supported", parentType)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &webhook{}
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	res, err := mapToWebhook(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map webhook to external type: %w", err)
	}

	return res, nil
}

// Create creates a new webhook.
func (s *WebhookStore) Create(ctx context.Context, hook *types.Webhook) error {
	const sqlQuery = `
		INSERT INTO webhooks (
			webhook_repo_id
			,webhook_space_id
			,webhook_created_by
			,webhook_created
			,webhook_updated
			,webhook_uid
			,webhook_display_name
			,webhook_description
			,webhook_url
			,webhook_secret
			,webhook_enabled
			,webhook_insecure
			,webhook_triggers
			,webhook_latest_execution_result
			,webhook_internal
		) values (
			:webhook_repo_id
			,:webhook_space_id
			,:webhook_created_by
			,:webhook_created
			,:webhook_updated
			,:webhook_uid
			,:webhook_display_name
			,:webhook_description
			,:webhook_url
			,:webhook_secret
			,:webhook_enabled
			,:webhook_insecure
			,:webhook_triggers
			,:webhook_latest_execution_result
			,:webhook_internal
		) RETURNING webhook_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbHook, err := mapToInternalWebhook(hook)
	if err != nil {
		return fmt.Errorf("failed to map webhook to internal db type: %w", err)
	}

	query, arg, err := db.BindNamed(sqlQuery, dbHook)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind webhook object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&hook.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// Update updates an existing webhook.
func (s *WebhookStore) Update(ctx context.Context, hook *types.Webhook) error {
	const sqlQuery = `
		UPDATE webhooks
		SET
			 webhook_version = :webhook_version
			,webhook_updated = :webhook_updated
			,webhook_uid = :webhook_uid
			,webhook_display_name = :webhook_display_name
			,webhook_description = :webhook_description
			,webhook_url = :webhook_url
			,webhook_secret = :webhook_secret
			,webhook_enabled = :webhook_enabled
			,webhook_insecure = :webhook_insecure
			,webhook_triggers = :webhook_triggers
			,webhook_latest_execution_result = :webhook_latest_execution_result
			,webhook_internal = :webhook_internal
		WHERE webhook_id = :webhook_id and webhook_version = :webhook_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	dbHook, err := mapToInternalWebhook(hook)
	if err != nil {
		return fmt.Errorf("failed to map webhook to internal db type: %w", err)
	}

	// update Version (used for optimistic locking) and Updated time
	dbHook.Version++
	dbHook.Updated = time.Now().UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbHook)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind webhook object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to update webhook")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	hook.Version = dbHook.Version
	hook.Updated = dbHook.Updated

	return nil
}

// UpdateOptLock updates the webhook using the optimistic locking mechanism.
func (s *WebhookStore) UpdateOptLock(ctx context.Context, hook *types.Webhook,
	mutateFn func(hook *types.Webhook) error) (*types.Webhook, error) {
	for {
		dup := *hook

		err := mutateFn(&dup)
		if err != nil {
			return nil, fmt.Errorf("failed to mutate the webhook: %w", err)
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, fmt.Errorf("failed to update the webhook: %w", err)
		}

		hook, err = s.Find(ctx, hook.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to find the latst version of the webhook: %w", err)
		}
	}
}

// Delete deletes the webhook for the given id.
func (s *WebhookStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM webhooks
		WHERE webhook_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// DeleteByIdentifier deletes the webhook with the given Identifier for the given parent.
func (s *WebhookStore) DeleteByIdentifier(
	ctx context.Context,
	parentType enum.WebhookParent,
	parentID int64,
	identifier string,
) error {
	stmt := database.Builder.
		Delete("webhooks").
		Where("LOWER(webhook_uid) = ?", strings.ToLower(identifier))

	switch parentType {
	case enum.WebhookParentRepo:
		stmt = stmt.Where("webhook_repo_id = ?", parentID)
	case enum.WebhookParentSpace:
		stmt = stmt.Where("webhook_space_id = ?", parentID)
	default:
		return fmt.Errorf("webhook parent type '%s' is not supported", parentType)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// Count counts the webhooks for a given parent type and id.
func (s *WebhookStore) Count(
	ctx context.Context,
	parents []types.WebhookParentInfo,
	opts *types.WebhookFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("webhooks")

	err := selectParents(parents, &stmt)
	if err != nil {
		return 0, fmt.Errorf("failed to select parents: %w", err)
	}

	stmt = applyWebhookFilter(opts, stmt)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (s *WebhookStore) List(
	ctx context.Context,
	parents []types.WebhookParentInfo,
	opts *types.WebhookFilter,
) ([]*types.Webhook, error) {
	stmt := database.Builder.
		Select(webhookColumns).
		From("webhooks")

	err := selectParents(parents, &stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to select parents: %w", err)
	}

	stmt = applyWebhookFilter(opts, stmt)

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	switch opts.Sort {
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed
	case enum.WebhookAttrID, enum.WebhookAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("webhook_id " + opts.Order.String())

		// TODO [CODE-1363]: remove after identifier migration.
	case enum.WebhookAttrUID, enum.WebhookAttrIdentifier:
		stmt = stmt.OrderBy("LOWER(webhook_uid) " + opts.Order.String())
		// TODO [CODE-1364]: Remove once UID/Identifier migration is completed
	case enum.WebhookAttrDisplayName:
		stmt = stmt.OrderBy("webhook_display_name " + opts.Order.String())
		//TODO: Postgres does not support COLLATE NOCASE for UTF8
	case enum.WebhookAttrCreated:
		stmt = stmt.OrderBy("webhook_created " + opts.Order.String())
	case enum.WebhookAttrUpdated:
		stmt = stmt.OrderBy("webhook_updated " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*webhook{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	res, err := mapToWebhooks(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map webhooks to external type: %w", err)
	}

	return res, nil
}

func mapToWebhook(hook *webhook) (*types.Webhook, error) {
	res := &types.Webhook{
		ID:         hook.ID,
		Version:    hook.Version,
		CreatedBy:  hook.CreatedBy,
		Created:    hook.Created,
		Updated:    hook.Updated,
		Identifier: hook.Identifier,
		// TODO [CODE-1364]: Remove once UID/Identifier migration is completed
		DisplayName:           hook.DisplayName,
		Description:           hook.Description,
		URL:                   hook.URL,
		Secret:                hook.Secret,
		Enabled:               hook.Enabled,
		Insecure:              hook.Insecure,
		Triggers:              triggersFromString(hook.Triggers),
		LatestExecutionResult: (*enum.WebhookExecutionResult)(hook.LatestExecutionResult.Ptr()),
		Internal:              hook.Internal,
	}

	switch {
	case hook.RepoID.Valid && hook.SpaceID.Valid:
		return nil, fmt.Errorf("both repoID and spaceID are set for hook %d", hook.ID)
	case hook.RepoID.Valid:
		res.ParentType = enum.WebhookParentRepo
		res.ParentID = hook.RepoID.Int64
	case hook.SpaceID.Valid:
		res.ParentType = enum.WebhookParentSpace
		res.ParentID = hook.SpaceID.Int64
	default:
		return nil, fmt.Errorf("neither repoID nor spaceID are set for hook %d", hook.ID)
	}

	return res, nil
}

func mapToInternalWebhook(hook *types.Webhook) (*webhook, error) {
	res := &webhook{
		ID:         hook.ID,
		Version:    hook.Version,
		CreatedBy:  hook.CreatedBy,
		Created:    hook.Created,
		Updated:    hook.Updated,
		Identifier: hook.Identifier,
		// TODO [CODE-1364]: Remove once UID/Identifier migration is completed
		DisplayName:           hook.DisplayName,
		Description:           hook.Description,
		URL:                   hook.URL,
		Secret:                hook.Secret,
		Enabled:               hook.Enabled,
		Insecure:              hook.Insecure,
		Triggers:              triggersToString(hook.Triggers),
		LatestExecutionResult: null.StringFromPtr((*string)(hook.LatestExecutionResult)),
		Internal:              hook.Internal,
	}

	switch hook.ParentType {
	case enum.WebhookParentRepo:
		res.RepoID = null.IntFrom(hook.ParentID)
	case enum.WebhookParentSpace:
		res.SpaceID = null.IntFrom(hook.ParentID)
	default:
		return nil, fmt.Errorf("webhook parent type %q is not supported", hook.ParentType)
	}

	return res, nil
}

func mapToWebhooks(hooks []*webhook) ([]*types.Webhook, error) {
	var err error
	m := make([]*types.Webhook, len(hooks))
	for i, hook := range hooks {
		m[i], err = mapToWebhook(hook)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// triggersSeparator defines the character that's used to join triggers for storing them in the DB
// ASSUMPTION: triggers are defined in an enum and don't contain ",".
const triggersSeparator = ","

func triggersFromString(triggersString string) []enum.WebhookTrigger {
	if triggersString == "" {
		return []enum.WebhookTrigger{}
	}

	rawTriggers := strings.Split(triggersString, triggersSeparator)

	triggers := make([]enum.WebhookTrigger, len(rawTriggers))
	for i, rawTrigger := range rawTriggers {
		// ASSUMPTION: trigger is valid value (as we wrote it to DB)
		triggers[i] = enum.WebhookTrigger(rawTrigger)
	}

	return triggers
}

func triggersToString(triggers []enum.WebhookTrigger) string {
	rawTriggers := make([]string, len(triggers))
	for i := range triggers {
		rawTriggers[i] = string(triggers[i])
	}

	return strings.Join(rawTriggers, triggersSeparator)
}

func applyWebhookFilter(
	opts *types.WebhookFilter,
	stmt squirrel.SelectBuilder,
) squirrel.SelectBuilder {
	if opts.Query != "" {
		stmt = stmt.Where("LOWER(webhook_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	if opts.SkipInternal {
		stmt = stmt.Where("webhook_internal != ?", true)
	}

	return stmt
}

func selectParents(
	parents []types.WebhookParentInfo,
	stmt *squirrel.SelectBuilder,
) error {
	var parentSelector squirrel.Or
	for _, parent := range parents {
		switch parent.Type {
		case enum.WebhookParentRepo:
			parentSelector = append(parentSelector, squirrel.Eq{
				"webhook_repo_id": parent.ID,
			})
		case enum.WebhookParentSpace:
			parentSelector = append(parentSelector, squirrel.Eq{
				"webhook_space_id": parent.ID,
			})
		default:
			return fmt.Errorf("webhook parent type '%s' is not supported", parent.Type)
		}
	}

	*stmt = stmt.Where(parentSelector)

	return nil
}
