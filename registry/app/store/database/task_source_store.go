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
	"time"

	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type taskSourceStore struct {
	db *sqlx.DB
	tx dbtx.Transactor
}

// NewTaskSourceStore returns a new TaskSourceRepository implementation.
func NewTaskSourceStore(db *sqlx.DB, tx dbtx.Transactor) store.TaskSourceRepository {
	return &taskSourceStore{
		db: db,
		tx: tx,
	}
}

// FindByTaskKeyAndSourceType returns a task source by its task key and source type.
func (s *taskSourceStore) FindByTaskKeyAndSourceType(
	ctx context.Context, key string, sourceType string,
) (*types.TaskSource, error) {
	stmt := databaseg.Builder.
		Select(
			"registry_task_source_key", "registry_task_source_type", "registry_task_source_id",
			"registry_task_source_error", "registry_task_source_run_id", "registry_task_source_updated_at").
		From("registry_task_sources").
		Where("registry_task_source_key = ?", key).
		Where("registry_task_source_type = ?", sourceType)

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	taskSourceDB := &TaskSourceDB{}
	if err := s.db.GetContext(ctx, taskSourceDB, query, args...); err != nil {
		return nil, fmt.Errorf("failed to find task with key %s: %w", key, err)
	}
	return taskSourceDB.ToTaskSource(), nil
}

// InsertSource inserts a source for a task.
func (s *taskSourceStore) InsertSource(ctx context.Context, key string, source types.SourceRef) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Insert("registry_task_sources").
		Columns("registry_task_source_key", "registry_task_source_type",
			"registry_task_source_id", "registry_task_source_updated_at").
		Values(key, source.Type, source.ID, now).
		Suffix("ON CONFLICT (registry_task_source_key, registry_task_source_type, registry_task_source_id) "+
			"DO UPDATE SET registry_task_source_status = ? WHERE registry_task_sources.registry_task_source_status = ?",
			types.TaskStatusPending, types.TaskStatusFailure)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert source query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert source (type: %s, id: %d) for task key %s: %w",
			source.Type, source.ID, key, err)
	}
	return nil
}

// ClaimSources marks sources as processing for a specific run and returns them.
func (s *taskSourceStore) ClaimSources(ctx context.Context, key string, runID string) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Update("registry_task_sources").
		Set("registry_task_source_status", types.TaskStatusProcessing).
		Set("registry_task_source_run_id", runID).
		Set("registry_task_source_updated_at", now).
		Where("registry_task_source_key = ?", key).
		Where("registry_task_source_status = ?", types.TaskStatusPending)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build claim sources query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to claim pending sources for task %s: %w", key, err)
	}
	return nil
}

// UpdateSourceStatus updates the status of sources for a specific run.
func (s *taskSourceStore) UpdateSourceStatus(
	ctx context.Context,
	runID string,
	status types.TaskStatus,
	errMsg string,
) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Update("registry_task_sources").
		Set("registry_task_source_status", status).
		Set("registry_task_source_updated_at", now)
	if errMsg == "" {
		stmt = stmt.Set("registry_task_source_error", nil)
	} else {
		stmt = stmt.Set("registry_task_source_error", errMsg)
	}
	stmt = stmt.Where("registry_task_source_run_id = ?", runID)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update source status query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update source status for run ID %s to %s: %w", runID, status, err)
	}
	return nil
}

// TaskSourceDB represents a database entity for task source processing.
type TaskSourceDB struct {
	Key       string  `db:"registry_task_source_key"`
	SrcType   string  `db:"registry_task_source_type"`
	SrcID     int64   `db:"registry_task_source_id"`
	Status    string  `db:"registry_task_source_status"`
	RunID     *string `db:"registry_task_source_run_id"`
	Error     *string `db:"registry_task_source_error"`
	UpdatedAt int64   `db:"registry_task_source_updated_at"`
}

// ToTaskSource converts a database task source entity to a DTO.
func (s *TaskSourceDB) ToTaskSource() *types.TaskSource {
	return &types.TaskSource{
		Key:       s.Key,
		SrcType:   types.SourceType(s.SrcType),
		SrcID:     s.SrcID,
		Status:    types.TaskStatus(s.Status),
		RunID:     s.RunID,
		Error:     s.Error,
		UpdatedAt: time.UnixMilli(s.UpdatedAt),
	}
}

// ToSourceRef converts a TaskSourceDB to a SourceRef.
func (s *TaskSourceDB) ToSourceRef() types.SourceRef {
	return types.SourceRef{
		Type: types.SourceType(s.SrcType),
		ID:   s.SrcID,
	}
}
