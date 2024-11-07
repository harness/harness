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
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
)

var _ store.TriggerStore = (*triggerStore)(nil)

type trigger struct {
	ID          int64              `db:"trigger_id"`
	Identifier  string             `db:"trigger_uid"`
	Description string             `db:"trigger_description"`
	Type        string             `db:"trigger_type"`
	Secret      string             `db:"trigger_secret"`
	PipelineID  int64              `db:"trigger_pipeline_id"`
	RepoID      int64              `db:"trigger_repo_id"`
	CreatedBy   int64              `db:"trigger_created_by"`
	Disabled    bool               `db:"trigger_disabled"`
	Actions     sqlxtypes.JSONText `db:"trigger_actions"`
	Created     int64              `db:"trigger_created"`
	Updated     int64              `db:"trigger_updated"`
	Version     int64              `db:"trigger_version"`
}

func mapInternalToTrigger(trigger *trigger) (*types.Trigger, error) {
	var actions []enum.TriggerAction
	err := json.Unmarshal(trigger.Actions, &actions)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal trigger.actions")
	}

	return &types.Trigger{
		ID:          trigger.ID,
		Description: trigger.Description,
		Type:        trigger.Type,
		Secret:      trigger.Secret,
		PipelineID:  trigger.PipelineID,
		RepoID:      trigger.RepoID,
		CreatedBy:   trigger.CreatedBy,
		Disabled:    trigger.Disabled,
		Actions:     actions,
		Identifier:  trigger.Identifier,
		Created:     trigger.Created,
		Updated:     trigger.Updated,
		Version:     trigger.Version,
	}, nil
}

func mapInternalToTriggerList(triggers []*trigger) ([]*types.Trigger, error) {
	ret := make([]*types.Trigger, len(triggers))
	for i, t := range triggers {
		trigger, err := mapInternalToTrigger(t)
		if err != nil {
			return nil, err
		}
		ret[i] = trigger
	}
	return ret, nil
}

func mapTriggerToInternal(t *types.Trigger) *trigger {
	return &trigger{
		ID:          t.ID,
		Identifier:  t.Identifier,
		Description: t.Description,
		Type:        t.Type,
		PipelineID:  t.PipelineID,
		Secret:      t.Secret,
		RepoID:      t.RepoID,
		CreatedBy:   t.CreatedBy,
		Disabled:    t.Disabled,
		Actions:     EncodeToSQLXJSON(t.Actions),
		Created:     t.Created,
		Updated:     t.Updated,
		Version:     t.Version,
	}
}

// NewTriggerStore returns a new TriggerStore.
func NewTriggerStore(db *sqlx.DB) store.TriggerStore {
	return &triggerStore{
		db: db,
	}
}

type triggerStore struct {
	db *sqlx.DB
}

const (
	triggerColumns = `
		trigger_id
		,trigger_uid
		,trigger_disabled
		,trigger_actions
		,trigger_description
		,trigger_pipeline_id
		,trigger_created
		,trigger_updated
		,trigger_version
	`
)

// Find returns an trigger given a pipeline ID and a trigger identifier.
func (s *triggerStore) FindByIdentifier(
	ctx context.Context,
	pipelineID int64,
	identifier string,
) (*types.Trigger, error) {
	const findQueryStmt = `
	SELECT` + triggerColumns + `
	FROM triggers
	WHERE trigger_pipeline_id = $1 AND trigger_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(trigger)
	if err := db.GetContext(ctx, dst, findQueryStmt, pipelineID, identifier); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find trigger")
	}
	return mapInternalToTrigger(dst)
}

// Create creates a new trigger in the datastore.
func (s *triggerStore) Create(ctx context.Context, t *types.Trigger) error {
	const triggerInsertStmt = `
	INSERT INTO triggers (
		trigger_uid
		,trigger_description
		,trigger_actions
		,trigger_disabled
		,trigger_type
		,trigger_secret
		,trigger_created_by
		,trigger_pipeline_id
		,trigger_repo_id
		,trigger_created
		,trigger_updated
		,trigger_version
	) VALUES (
		:trigger_uid
		,:trigger_description
		,:trigger_actions
		,:trigger_disabled
		,:trigger_type
		,:trigger_secret
		,:trigger_created_by
		,:trigger_pipeline_id
		,:trigger_repo_id
		,:trigger_created
		,:trigger_updated
		,:trigger_version
	) RETURNING trigger_id`
	db := dbtx.GetAccessor(ctx, s.db)

	trigger := mapTriggerToInternal(t)
	query, arg, err := db.BindNamed(triggerInsertStmt, trigger)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind trigger object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&trigger.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Trigger query failed")
	}

	return nil
}

// Update tries to update an trigger in the datastore with optimistic locking.
func (s *triggerStore) Update(ctx context.Context, t *types.Trigger) error {
	const triggerUpdateStmt = `
	UPDATE triggers
	SET
		trigger_uid = :trigger_uid
		,trigger_description = :trigger_description
		,trigger_disabled = :trigger_disabled
		,trigger_updated = :trigger_updated
		,trigger_actions = :trigger_actions
		,trigger_version = :trigger_version
	WHERE trigger_id = :trigger_id AND trigger_version = :trigger_version - 1`
	updatedAt := time.Now()
	trigger := mapTriggerToInternal(t)

	trigger.Version++
	trigger.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(triggerUpdateStmt, trigger)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind trigger object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update trigger")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	t.Version = trigger.Version
	t.Updated = trigger.Updated
	return nil
}

// UpdateOptLock updates the pipeline using the optimistic locking mechanism.
func (s *triggerStore) UpdateOptLock(ctx context.Context,
	trigger *types.Trigger,
	mutateFn func(trigger *types.Trigger) error) (*types.Trigger, error) {
	for {
		dup := *trigger

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		trigger, err = s.FindByIdentifier(ctx, trigger.PipelineID, trigger.Identifier)
		if err != nil {
			return nil, err
		}
	}
}

// List lists the triggers for a given pipeline ID.
func (s *triggerStore) List(
	ctx context.Context,
	pipelineID int64,
	filter types.ListQueryFilter,
) ([]*types.Trigger, error) {
	stmt := database.Builder.
		Select(triggerColumns).
		From("triggers").
		Where("trigger_pipeline_id = ?", fmt.Sprint(pipelineID))

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("trigger_uid", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*trigger{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return mapInternalToTriggerList(dst)
}

// ListAllEnabled lists all enabled triggers for a given repo without pagination.
func (s *triggerStore) ListAllEnabled(
	ctx context.Context,
	repoID int64,
) ([]*types.Trigger, error) {
	stmt := database.Builder.
		Select(triggerColumns).
		From("triggers").
		Where("trigger_repo_id = ? AND trigger_disabled = false", fmt.Sprint(repoID))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*trigger{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return mapInternalToTriggerList(dst)
}

// Count of triggers under a given pipeline.
func (s *triggerStore) Count(ctx context.Context, pipelineID int64, filter types.ListQueryFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("triggers").
		Where("trigger_pipeline_id = ?", pipelineID)

	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("trigger_uid", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

// Delete deletes an trigger given a pipeline ID and a trigger identifier.
func (s *triggerStore) DeleteByIdentifier(ctx context.Context, pipelineID int64, identifier string) error {
	const triggerDeleteStmt = `
		DELETE FROM triggers
		WHERE trigger_pipeline_id = $1 AND trigger_uid = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, triggerDeleteStmt, pipelineID, identifier); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete trigger")
	}

	return nil
}
