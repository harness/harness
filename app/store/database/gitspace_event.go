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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.GitspaceEventStore = (*gitspaceEventStore)(nil)

const (
	gitspaceEventIDColumn = `geven_id`
	gitspaceEventsColumns = `
		geven_event,
		geven_created,
		geven_entity_type,
		geven_entity_uid,
		geven_entity_id
	`
	gitspaceEventsColumnsWithID = gitspaceEventIDColumn + `,
		` + gitspaceEventsColumns
	gitspaceEventsTable = `gitspace_events`
)

type gitspaceEventStore struct {
	db *sqlx.DB
}

type gitspaceEvent struct {
	ID               int64                   `db:"geven_id"`
	Event            enum.GitspaceEventType  `db:"geven_event"`
	Created          int64                   `db:"geven_created"`
	EntityType       enum.GitspaceEntityType `db:"geven_entity_type"`
	EntityIdentifier string                  `db:"geven_entity_uid"`
	EntityID         int64                   `db:"geven_entity_id"`
}

func NewGitspaceEventStore(db *sqlx.DB) store.GitspaceEventStore {
	return &gitspaceEventStore{
		db: db,
	}
}

func (g gitspaceEventStore) FindLatestByTypeAndGitspaceConfigID(
	ctx context.Context,
	eventType enum.GitspaceEventType,
	gitspaceConfigID int64,
) (*types.GitspaceEvent, error) {
	stmt := database.Builder.
		Select(gitspaceEventsColumnsWithID).
		From(gitspaceEventsTable).
		Where("geven_event = $1", eventType).
		Where("geven_entity_id = $2", gitspaceConfigID).
		OrderBy("geven_created DESC")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert squirrel builder to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, g.db)
	gitspaceEventEntity := new(gitspaceEvent)
	if err = db.GetContext(ctx, gitspaceEventEntity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspaceEvent")
	}
	return g.mapGitspaceEvent(gitspaceEventEntity), nil
}

func (g gitspaceEventStore) Create(ctx context.Context, gitspaceEvent *types.GitspaceEvent) error {
	stmt := database.Builder.
		Insert(gitspaceEventsTable).
		Columns(gitspaceEventsColumns).
		Values(
			gitspaceEvent.Event,
			gitspaceEvent.Created,
			gitspaceEvent.EntityType,
			gitspaceEvent.EntityIdentifier,
			gitspaceEvent.EntityID,
		).
		Suffix("RETURNING " + gitspaceEventIDColumn)
	db := dbtx.GetAccessor(ctx, g.db)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert squirrel builder to sql: %w", err)
	}
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&gitspaceEvent.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "%s query failed", gitspaceEventsTable)
	}
	return nil
}

func (g gitspaceEventStore) List(
	ctx context.Context,
	filter *types.GitspaceEventFilter,
) ([]*types.GitspaceEvent, error) {
	stmt := database.Builder.
		Select(gitspaceEventsColumnsWithID).
		From(gitspaceEventsTable).
		Where("geven_entity_id = $1", filter.EntityID).
		Where("geven_entity_type = $2", filter.EntityType)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert squirrel builder to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, g.db)
	gitspaceEventEntities := make([]*gitspaceEvent, 0)
	if err = db.SelectContext(ctx, gitspaceEventEntities, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspaceEvent")
	}
	gitspaceEvents := g.mapGitspaceEvents(gitspaceEventEntities)
	return gitspaceEvents, nil
}

func (g gitspaceEventStore) mapGitspaceEvents(gitspaceEventEntities []*gitspaceEvent) []*types.GitspaceEvent {
	gitspaceEvents := make([]*types.GitspaceEvent, len(gitspaceEventEntities))
	for index, gitspaceEventEntity := range gitspaceEventEntities {
		currentEntity := gitspaceEventEntity
		gitspaceEvents[index] = g.mapGitspaceEvent(currentEntity)
	}
	return gitspaceEvents
}

func (g gitspaceEventStore) mapGitspaceEvent(event *gitspaceEvent) *types.GitspaceEvent {
	return &types.GitspaceEvent{
		Event:            event.Event,
		Created:          event.Created,
		EntityType:       event.EntityType,
		EntityIdentifier: event.EntityIdentifier,
		EntityID:         event.EntityID,
	}
}
