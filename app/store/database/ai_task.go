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
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	aiTaskTable         = `ai_tasks`
	aiTaskInsertColumns = `
		aitask_uid,
		aitask_gitspace_config_id,
		aitask_gitspace_instance_id,
		aitask_initial_prompt,
		aitask_display_name,
		aitask_user_uid,
		aitask_space_id,
		aitask_created,
		aitask_updated,
		aitask_api_url,
		aitask_ai_agent,
		aitask_state,
		aitask_output`
	aiTaskSelectColumns = "aitask_id," + aiTaskInsertColumns
)

type aiTask struct {
	ID                 int64            `db:"aitask_id"`
	Identifier         string           `db:"aitask_uid"`
	GitspaceConfigID   int64            `db:"aitask_gitspace_config_id"`
	GitspaceInstanceID int64            `db:"aitask_gitspace_instance_id"`
	InitialPrompt      string           `db:"aitask_initial_prompt"`
	DisplayName        string           `db:"aitask_display_name"`
	UserUID            string           `db:"aitask_user_uid"`
	SpaceID            int64            `db:"aitask_space_id"`
	Created            int64            `db:"aitask_created"`
	Updated            int64            `db:"aitask_updated"`
	APIURL             null.String      `db:"aitask_api_url"`
	AgentType          enum.AIAgent     `db:"aitask_ai_agent"`
	State              enum.AITaskState `db:"aitask_state"`
	Output             null.String      `db:"aitask_output"`
}

var _ store.AITaskStore = (*aiTaskStore)(nil)

// NewAITaskStore returns a new AITaskStore.
func NewAITaskStore(db *sqlx.DB) store.AITaskStore {
	return &aiTaskStore{
		db: db,
	}
}

type aiTaskStore struct {
	db *sqlx.DB
}

func (s aiTaskStore) Create(ctx context.Context, aiTask *types.AITask) error {
	stmt := database.Builder.
		Insert(aiTaskTable).
		Columns(aiTaskInsertColumns).
		Values(
			aiTask.Identifier,
			aiTask.GitspaceConfigID,
			aiTask.GitspaceInstanceID,
			aiTask.InitialPrompt,
			aiTask.DisplayName,
			aiTask.UserUID,
			aiTask.SpaceID,
			aiTask.Created,
			aiTask.Updated,
			aiTask.APIURL,
			aiTask.AIAgent,
			aiTask.State,
			aiTask.Output,
		).
		Suffix("RETURNING aitask_id")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&aiTask.ID); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "ai task create query failed for %s", aiTask.Identifier)
	}
	return nil
}
func (s aiTaskStore) Update(ctx context.Context, aiTask *types.AITask) error {
	stmt := database.Builder.
		Update(aiTaskTable).
		Set("aitask_display_name", aiTask.DisplayName).
		Set("aitask_updated", aiTask.Updated).
		Set("aitask_api_url", aiTask.APIURL).
		Set("aitask_state", aiTask.State).
		Set("aitask_output", aiTask.Output).
		Where("aitask_id = ?", aiTask.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "Failed to update ai task for %s", aiTask.Identifier)
	}
	return nil
}
func (s aiTaskStore) Find(ctx context.Context, id int64) (*types.AITask, error) {
	stmt := database.Builder.
		Select(aiTaskSelectColumns).
		From(aiTaskTable).
		Where("aitask_id = ?", id)
	dst := new(aiTask)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find ai task for %d", id)
	}
	return s.mapDBToAITask(dst), nil
}
func (s aiTaskStore) FindByIdentifier(ctx context.Context, identifier string) (*types.AITask, error) {
	stmt := database.Builder.
		Select(aiTaskSelectColumns).
		From(aiTaskTable).
		Where("LOWER(aitask_uid) = ?", strings.ToLower(identifier))
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(aiTask)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find ai task for %s", identifier)
	}
	return s.mapDBToAITask(dst), nil
}
func (s aiTaskStore) List(ctx context.Context, filter *types.AITaskFilter) ([]*types.AITask, error) {
	stmt := database.Builder.
		Select(aiTaskSelectColumns).
		From(aiTaskTable)
	stmt = s.addAITaskFilter(stmt, filter)
	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))
	stmt = stmt.OrderBy("aitask_created DESC")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*aiTask
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing list ai task query")
	}
	return s.mapToAITasks(dst), nil
}
func (s aiTaskStore) addAITaskFilter(stmt squirrel.SelectBuilder, filter *types.AITaskFilter) squirrel.SelectBuilder {
	if filter.SpaceID != 0 {
		stmt = stmt.Where("aitask_space_id = ?", filter.SpaceID)
	}
	if filter.GitspaceConfigID != 0 {
		stmt = stmt.Where("aitask_gitspace_config_id = ?", filter.GitspaceConfigID)
	}
	if filter.GitspaceInstanceID != 0 {
		stmt = stmt.Where("aitask_gitspace_instance_id = ?", filter.GitspaceInstanceID)
	}
	if filter.AIAgent != "" {
		stmt = stmt.Where("aitask_ai_agent = ?", filter.AIAgent)
	}
	if filter.State != "" {
		stmt = stmt.Where("aitask_state = ?", filter.State)
	}
	if filter.Query != "" {
		stmt = stmt.Where(squirrel.Or{
			squirrel.Expr("LOWER(aitask_uid) LIKE ?", "%"+strings.ToLower(filter.Query)+"%"),
			squirrel.Expr("LOWER(aitask_display_name) LIKE ?", "%"+strings.ToLower(filter.Query)+"%"),
		})
	}
	return stmt
}
func (s aiTaskStore) mapDBToAITask(in *aiTask) *types.AITask {
	return &types.AITask{
		ID:                 in.ID,
		Identifier:         in.Identifier,
		GitspaceConfigID:   in.GitspaceConfigID,
		GitspaceInstanceID: in.GitspaceInstanceID,
		InitialPrompt:      in.InitialPrompt,
		DisplayName:        in.DisplayName,
		UserUID:            in.UserUID,
		SpaceID:            in.SpaceID,
		Created:            in.Created,
		Updated:            in.Updated,
		APIURL:             in.APIURL.Ptr(),
		AIAgent:            in.AgentType,
		State:              in.State,
		Output:             in.Output.Ptr(),
	}
}
func (s aiTaskStore) mapToAITasks(aiTasks []*aiTask) []*types.AITask {
	res := make([]*types.AITask, len(aiTasks))
	for i := range aiTasks {
		res[i] = s.mapDBToAITask(aiTasks[i])
	}
	return res
}
