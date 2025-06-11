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
	databaseg "github.com/harness/gitness/store/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type taskEventStore struct {
	db *sqlx.DB
}

// NewTaskEventStore returns a new TaskEventRepository implementation.
func NewTaskEventStore(db *sqlx.DB) store.TaskEventRepository {
	return &taskEventStore{
		db: db,
	}
}

// LogTaskEvent logs an event in the task_events table.
func (s *taskEventStore) LogTaskEvent(ctx context.Context, key string, event string, payload []byte) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Insert("registry_task_events").
		Columns("registry_task_event_id", "registry_task_event_key", "registry_task_event_event",
			"registry_task_event_payload", "registry_task_event_created_at").
		Values(uuid.NewString(), key, event, payload, now)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert event query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to log task event '%s' for task %s: %w", event, key, err)
	}
	return nil
}
