// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.TriggerStore = (*triggerStore)(nil)

// NewTriggerStore returns a new TriggerStore.
func NewTriggerStore(db *sqlx.DB) *triggerStore {
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
		,trigger_description
		,trigger_pipeline_id
		,trigger_created
		,trigger_updated
		,trigger_version
	`
)

// Find returns an trigger given a pipeline ID and a trigger UID.
func (s *triggerStore) FindByUID(ctx context.Context, pipelineID int64, uid string) (*types.Trigger, error) {
	const findQueryStmt = `
	SELECT` + triggerColumns + `
	FROM triggers
	WHERE trigger_pipeline_id = $1 AND trigger_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Trigger)
	if err := db.GetContext(ctx, dst, findQueryStmt, pipelineID, uid); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find trigger")
	}
	return dst, nil
}

// Create creates a new trigger in the datastore.
func (s *triggerStore) Create(ctx context.Context, trigger *types.Trigger) error {
	const triggerInsertStmt = `
	INSERT INTO triggers (
		,trigger_uid
		,trigger_description
		,trigger_pipeline_id
		,trigger_created
		,trigger_updated
		,trigger_version
	) VALUES (
		,:trigger_uid
		,:trigger_description
		,:trigger_pipeline_id
		,:trigger_created
		,:trigger_updated
		,:trigger_version
	) RETURNING trigger_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(triggerInsertStmt, trigger)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind trigger object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&trigger.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Trigger query failed")
	}

	return nil
}

// Update tries to update an trigger in the datastore with optimistic locking.
func (s *triggerStore) Update(ctx context.Context, e *types.Trigger) error {
	const triggerUpdateStmt = `
	UPDATE triggers
	SET
		trigger_uid = :trigger_uid
		,trigger_description = :trigger_description
		,trigger_updated = :trigger_updated
		,trigger_version = :trigger_version
	WHERE trigger_id = :trigger_id AND trigger_version = :trigger_version - 1`
	updatedAt := time.Now()
	trigger := *e

	trigger.Version++
	trigger.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(triggerUpdateStmt, trigger)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind trigger object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update trigger")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	e.Version = trigger.Version
	e.Updated = trigger.Updated
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

		trigger, err = s.FindByUID(ctx, trigger.PipelineID, trigger.UID)
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
		stmt = stmt.Where("LOWER(trigger_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Trigger{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Count of triggers under a given pipeline.
func (s *triggerStore) Count(ctx context.Context, pipelineID int64, filter types.ListQueryFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("triggers").
		Where("trigger_pipeline_id = ?", pipelineID)

	if filter.Query != "" {
		stmt = stmt.Where("trigger_uid LIKE ?", fmt.Sprintf("%%%s%%", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// Delete deletes an trigger given a pipeline ID and a trigger UID.
func (s *triggerStore) DeleteByUID(ctx context.Context, pipelineID int64, uid string) error {
	const triggerDeleteStmt = `
		DELETE FROM triggers
		WHERE trigger_pipeline_id = $1 AND trigger_uid = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, triggerDeleteStmt, pipelineID, uid); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete trigger")
	}

	return nil
}
