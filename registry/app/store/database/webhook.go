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
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/types/enum"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const triggersSeparator = ","

var registryWebhooksFields = []string{
	"registry_webhook_id",
	"registry_webhook_version",
	"registry_webhook_registry_id",
	"registry_webhook_space_id",
	"registry_webhook_created_by",
	"registry_webhook_created",
	"registry_webhook_updated",
	"registry_webhook_scope",
	"registry_webhook_identifier",
	"registry_webhook_name",
	"registry_webhook_description",
	"registry_webhook_url",
	"registry_webhook_secret_identifier",
	"registry_webhook_secret_space_id",
	"registry_webhook_enabled",
	"registry_webhook_internal",
	"registry_webhook_insecure",
	"registry_webhook_triggers",
	"registry_webhook_extra_headers",
	"registry_webhook_latest_execution_result",
}

func NewWebhookDao(db *sqlx.DB) store.WebhooksRepository {
	return &WebhookDao{
		db: db,
	}
}

type webhookDB struct {
	ID                    int64          `db:"registry_webhook_id"`
	Version               int64          `db:"registry_webhook_version"`
	RegistryID            null.Int       `db:"registry_webhook_registry_id"`
	SpaceID               null.Int       `db:"registry_webhook_space_id"`
	CreatedBy             int64          `db:"registry_webhook_created_by"`
	Created               int64          `db:"registry_webhook_created"`
	Updated               int64          `db:"registry_webhook_updated"`
	Scope                 int64          `db:"registry_webhook_scope"`
	Internal              bool           `db:"registry_webhook_internal"`
	Identifier            string         `db:"registry_webhook_identifier"`
	Name                  string         `db:"registry_webhook_name"`
	Description           string         `db:"registry_webhook_description"`
	URL                   string         `db:"registry_webhook_url"`
	SecretIdentifier      sql.NullString `db:"registry_webhook_secret_identifier"`
	SecretSpaceID         sql.NullInt64  `db:"registry_webhook_secret_space_id"`
	Enabled               bool           `db:"registry_webhook_enabled"`
	Insecure              bool           `db:"registry_webhook_insecure"`
	Triggers              string         `db:"registry_webhook_triggers"`
	ExtraHeaders          null.String    `db:"registry_webhook_extra_headers"`
	LatestExecutionResult null.String    `db:"registry_webhook_latest_execution_result"`
}

type WebhookDao struct {
	db *sqlx.DB
}

func (w WebhookDao) Create(ctx context.Context, webhook *types.Webhook) error {
	const sqlQuery = `
		INSERT INTO registry_webhooks (
			registry_webhook_registry_id
			,registry_webhook_space_id
			,registry_webhook_created_by
			,registry_webhook_created
			,registry_webhook_updated
			,registry_webhook_identifier
			,registry_webhook_name
			,registry_webhook_description
			,registry_webhook_url
			,registry_webhook_secret_identifier
			,registry_webhook_secret_space_id
			,registry_webhook_enabled
			,registry_webhook_internal
			,registry_webhook_insecure
			,registry_webhook_triggers
			,registry_webhook_latest_execution_result
			,registry_webhook_extra_headers
			,registry_webhook_scope
		) values (
			:registry_webhook_registry_id
			,:registry_webhook_space_id
			,:registry_webhook_created_by
			,:registry_webhook_created
			,:registry_webhook_updated
			,:registry_webhook_identifier
			,:registry_webhook_name
			,:registry_webhook_description
			,:registry_webhook_url
			,:registry_webhook_secret_identifier
			,:registry_webhook_secret_space_id
			,:registry_webhook_enabled
			,:registry_webhook_internal
			,:registry_webhook_insecure
			,:registry_webhook_triggers
			,:registry_webhook_latest_execution_result
			,:registry_webhook_extra_headers
			,:registry_webhook_scope
		) RETURNING registry_webhook_id`

	db := dbtx.GetAccessor(ctx, w.db)

	dbwebhook, err := mapToWebhookDB(webhook)
	dbwebhook.Created = webhook.CreatedAt.UnixMilli()
	dbwebhook.Updated = webhook.UpdatedAt.UnixMilli()
	if err != nil {
		return fmt.Errorf("failed to map registry webhook to internal db type: %w", err)
	}

	query, arg, err := db.BindNamed(sqlQuery, dbwebhook)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to registry bind webhook object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&webhook.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

func (w WebhookDao) GetByRegistryAndIdentifier(
	ctx context.Context,
	registryID int64,
	webhookIdentifier string,
) (*types.Webhook, error) {
	query := database.Builder.Select(registryWebhooksFields...).
		From("registry_webhooks").
		Where("registry_webhook_registry_id = ? AND registry_webhook_identifier = ?", registryID, webhookIdentifier)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, w.db)

	dst := new(webhookDB)
	if err = db.GetContext(ctx, dst, sqlQuery, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to get webhook detail")
	}

	return mapToWebhook(dst)
}

func (w WebhookDao) ListByRegistry(
	ctx context.Context,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
	registryID int64,
) (*[]types.Webhook, error) {
	query := database.Builder.Select(registryWebhooksFields...).
		From("registry_webhooks").
		Where("registry_webhook_registry_id = ?", registryID)

	if search != "" {
		query = query.Where("registry_webhook_name LIKE ?", "%"+search+"%")
	}

	validSortFields := map[string]string{
		"name": "registry_webhook_name",
	}
	validSortByField := validSortFields[sortByField]
	if validSortByField != "" {
		query = query.OrderBy(fmt.Sprintf("%s %s", validSortByField, sortByOrder))
	}
	query = query.Limit(util.SafeIntToUInt64(limit)).Offset(util.SafeIntToUInt64(offset))

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, w.db)

	var dst []*webhookDB
	if err = db.SelectContext(ctx, &dst, sqlQuery, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list webhooks details")
	}

	return mapToWebhooksList(dst)
}

func (w WebhookDao) CountAllByRegistry(
	ctx context.Context,
	registryID int64,
	search string,
) (int64, error) {
	stmt := database.Builder.Select("COUNT(*)").
		From("registry_webhooks").
		Where("registry_webhook_registry_id = ?", registryID)

	if !commons.IsEmpty(search) {
		stmt = stmt.Where("registry_webhook_name LIKE ?", "%"+search+"%")
	}

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, w.db)

	var count int64
	err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (w WebhookDao) Update(ctx context.Context, webhook *types.Webhook) error {
	var sqlQuery = " UPDATE registry_webhooks SET " +
		util.GetSetDBKeys(webhookDB{},
			"registry_webhook_identifier",
			"registry_webhook_registry_id",
			"registry_webhook_created",
			"registry_webhook_created_by",
			"registry_webhook_version",
			"registry_webhook_internal") +
		", registry_webhook_version = registry_webhook_version + 1" +
		" WHERE registry_webhook_identifier = :registry_webhook_identifier" +
		" AND registry_webhook_registry_id = :registry_webhook_registry_id"

	dbWebhook, err := mapToWebhookDB(webhook)
	dbWebhook.Updated = webhook.UpdatedAt.UnixMilli()
	if err != nil {
		return err
	}
	dbWebhook.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, w.db)

	query, arg, err := db.BindNamed(sqlQuery, dbWebhook)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind registry webhook object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update registry webhook")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitnessstore.ErrVersionConflict
	}

	return nil
}

func (w WebhookDao) DeleteByRegistryAndIdentifier(
	ctx context.Context,
	registryID int64,
	webhookIdentifier string,
) error {
	sqlQuery := database.Builder.Delete("registry_webhooks").
		Where("registry_webhook_identifier = ? AND registry_webhook_registry_id = ?", webhookIdentifier, registryID)

	query, args, err := sqlQuery.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge registry_webhooks query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, w.db)

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete registry_webhooks query failed")
	}

	return nil
}

func mapToWebhookDB(webhook *types.Webhook) (*webhookDB, error) {
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = time.Now()
	}
	webhook.UpdatedAt = time.Now()
	dBwebhook := &webhookDB{
		ID:                    webhook.ID,
		Version:               webhook.Version,
		CreatedBy:             webhook.CreatedBy,
		Identifier:            webhook.Identifier,
		Scope:                 webhook.Scope,
		Name:                  webhook.Name,
		Description:           webhook.Description,
		URL:                   webhook.URL,
		SecretIdentifier:      util.GetEmptySQLString(webhook.SecretIdentifier),
		SecretSpaceID:         util.GetEmptySQLInt64(webhook.SecretSpaceID),
		Enabled:               webhook.Enabled,
		Insecure:              webhook.Insecure,
		Internal:              webhook.Internal,
		Triggers:              triggersToString(webhook.Triggers),
		ExtraHeaders:          null.StringFrom(structListToString(webhook.ExtraHeaders)),
		LatestExecutionResult: null.StringFromPtr((*string)(webhook.LatestExecutionResult)),
	}

	switch webhook.ParentType {
	case enum.WebhookParentRegistry:
		dBwebhook.RegistryID = null.IntFrom(webhook.ParentID)
	case enum.WebhookParentSpace:
		dBwebhook.SpaceID = null.IntFrom(webhook.ParentID)
	default:
		return nil, fmt.Errorf("webhook parent type %q is not supported", webhook.ParentType)
	}
	return dBwebhook, nil
}

func mapToWebhook(webhookDB *webhookDB) (*types.Webhook, error) {
	webhook := &types.Webhook{
		ID:                    webhookDB.ID,
		Version:               webhookDB.Version,
		CreatedBy:             webhookDB.CreatedBy,
		CreatedAt:             time.UnixMilli(webhookDB.Created),
		UpdatedAt:             time.UnixMilli(webhookDB.Updated),
		Scope:                 webhookDB.Scope,
		Identifier:            webhookDB.Identifier,
		Name:                  webhookDB.Name,
		Description:           webhookDB.Description,
		URL:                   webhookDB.URL,
		Enabled:               webhookDB.Enabled,
		Internal:              webhookDB.Internal,
		Insecure:              webhookDB.Insecure,
		Triggers:              triggersFromString(webhookDB.Triggers),
		ExtraHeaders:          stringToStructList(webhookDB.ExtraHeaders.String),
		LatestExecutionResult: (*artifact.WebhookExecResult)(webhookDB.LatestExecutionResult.Ptr()),
	}

	if webhookDB.SecretIdentifier.Valid {
		webhook.SecretIdentifier = webhookDB.SecretIdentifier.String
	}
	if webhookDB.SecretSpaceID.Valid {
		webhook.SecretSpaceID = int(webhookDB.SecretSpaceID.Int64)
	}

	switch {
	case webhookDB.RegistryID.Valid && webhookDB.SpaceID.Valid:
		return nil, fmt.Errorf("both registryID and spaceID are set for hook %d", webhookDB.ID)
	case webhookDB.RegistryID.Valid:
		webhook.ParentType = enum.WebhookParentRegistry
		webhook.ParentID = webhookDB.RegistryID.Int64
	case webhookDB.SpaceID.Valid:
		webhook.ParentType = enum.WebhookParentSpace
		webhook.ParentID = webhookDB.SpaceID.Int64
	default:
		return nil, fmt.Errorf("neither registryID nor spaceID are set for hook %d", webhookDB.ID)
	}

	return webhook, nil
}

func triggersToString(triggers []artifact.Trigger) string {
	rawTriggers := make([]string, len(triggers))
	for i := range triggers {
		rawTriggers[i] = string(triggers[i])
	}

	return strings.Join(rawTriggers, triggersSeparator)
}

func triggersFromString(triggersString string) []artifact.Trigger {
	if triggersString == "" {
		return []artifact.Trigger{}
	}

	rawTriggers := strings.Split(triggersString, triggersSeparator)
	triggers := make([]artifact.Trigger, len(rawTriggers))
	for i, rawTrigger := range rawTriggers {
		triggers[i] = artifact.Trigger(rawTrigger)
	}

	return triggers
}

// Convert a list of ExtraHeaders structs to a JSON string.
func structListToString(headers []artifact.ExtraHeader) string {
	jsonData, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// Convert a JSON string back to a list of ExtraHeaders structs.
func stringToStructList(jsonStr string) []artifact.ExtraHeader {
	var headers []artifact.ExtraHeader
	err := json.Unmarshal([]byte(jsonStr), &headers)
	if err != nil {
		return nil
	}
	return headers
}

func mapToWebhooksList(
	dst []*webhookDB,
) (*[]types.Webhook, error) {
	webhooks := make([]types.Webhook, 0, len(dst))
	for _, d := range dst {
		webhook, err := mapToWebhook(d)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, *webhook)
	}
	return &webhooks, nil
}
