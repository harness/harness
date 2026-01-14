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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.GitspaceSettingsStore = (*gitspaceSettingsStore)(nil)

const (
	gitspaceSettingsIDColumn = `gsett_id`
	gitspaceSettingsColumns  = `
		gsett_space_id,
		gsett_settings_data,
        gsett_settings_type,
        gsett_criteria_key,
		gsett_created,
		gsett_updated
	`
	gitspaceSettingsColumnsWithID = gitspaceSettingsIDColumn + `,
		` + gitspaceSettingsColumns
	gitspaceSettingsTable = `gitspace_settings`
)

type gitspaceSettingsStore struct {
	db *sqlx.DB
}

func (g gitspaceSettingsStore) List(
	ctx context.Context,
	spaceID int64,
	filter *types.GitspaceSettingsFilter,
) ([]*types.GitspaceSettings, error) {
	stmt := database.Builder.
		Select(gitspaceSettingsColumnsWithID).
		From(gitspaceSettingsTable).
		Where("gsett_space_id = $1", spaceID).
		OrderBy("gsett_updated DESC")

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert squirrel builder to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, g.db)
	var gitspaceSettingsEntity []*gitspaceSettings
	if err = db.SelectContext(ctx, &gitspaceSettingsEntity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(
			ctx, err, "Failed to find gitspace settings for space: %d", spaceID)
	}
	return g.mapToGitspaceSettings(gitspaceSettingsEntity)
}

func (g gitspaceSettingsStore) Upsert(ctx context.Context, in *types.GitspaceSettings) error {
	dbgitspaceSettings, err := g.mapToInternalGitspaceSettings(in)
	if err != nil {
		return err
	}
	stmt := database.Builder.
		Insert(gitspaceSettingsTable).
		Columns(gitspaceSettingsColumns).
		Values(
			dbgitspaceSettings.SpaceID,
			dbgitspaceSettings.SettingsData,
			dbgitspaceSettings.SettingsType,
			dbgitspaceSettings.CriteriaKey,
			dbgitspaceSettings.Created,
			dbgitspaceSettings.Updated).
		Suffix(`
ON CONFLICT (gsett_space_id, gsett_settings_type, gsett_criteria_key)
DO UPDATE 
SET 
	gsett_settings_data = EXCLUDED.gsett_settings_data,
	gsett_updated = EXCLUDED.gsett_updated`)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	if _, err = db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "gitspace settings upsert create query failed for %v %v", in.Settings, in.SpaceID)
	}
	return nil
}

func (g gitspaceSettingsStore) FindByType(
	ctx context.Context,
	spaceID int64,
	settingsType enum.GitspaceSettingsType,
	criteria *types.GitspaceSettingsCriteria,
) (*types.GitspaceSettings, error) {
	criteriaKey, err := criteria.ToKey()
	if err != nil {
		return nil, fmt.Errorf("failed to convert criteria to key: %w", err)
	}
	stmt := database.Builder.
		Select(gitspaceSettingsColumnsWithID).
		From(gitspaceSettingsTable).
		Where("gsett_settings_type = $1", settingsType).
		Where("gsett_space_id = $2", spaceID).
		Where("gsett_criteria_key = $3", criteriaKey).
		OrderBy("gsett_updated DESC")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert squirrel builder to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, g.db)
	gitspaceSettingsEntity := new(gitspaceSettings)
	if err = db.GetContext(ctx, gitspaceSettingsEntity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(
			ctx, err, "Failed to find gitspace settings for type:%v space: %d", settingsType, spaceID)
	}
	return g.mapGitspaceSettings(gitspaceSettingsEntity)
}

func NewGitspaceSettingsStore(db *sqlx.DB) store.GitspaceSettingsStore {
	return &gitspaceSettingsStore{
		db: db,
	}
}

type gitspaceSettings struct {
	ID           int64                     `db:"gsett_id"`
	SpaceID      int64                     `db:"gsett_space_id"`
	SettingsData []byte                    `db:"gsett_settings_data"`
	SettingsType enum.GitspaceSettingsType `db:"gsett_settings_type"`
	CriteriaKey  types.CriteriaKey         `db:"gsett_criteria_key"`
	Created      int64                     `db:"gsett_created"`
	Updated      int64                     `db:"gsett_updated"`
}

func (g gitspaceSettingsStore) mapGitspaceSettings(in *gitspaceSettings) (*types.GitspaceSettings, error) {
	var settingsData types.SettingsData
	if len(in.SettingsData) > 0 {
		marshalErr := json.Unmarshal(in.SettingsData, &settingsData)
		if marshalErr != nil {
			return nil, marshalErr
		}
	}
	return &types.GitspaceSettings{
		ID:           in.ID,
		SpaceID:      in.SpaceID,
		Settings:     settingsData,
		SettingsType: in.SettingsType,
		CriteriaKey:  in.CriteriaKey,
		Created:      in.Created,
		Updated:      in.Updated,
	}, nil
}

func (g gitspaceSettingsStore) mapToGitspaceSettings(
	in []*gitspaceSettings,
) ([]*types.GitspaceSettings, error) {
	var err error
	res := make([]*types.GitspaceSettings, len(in))
	for index := range in {
		res[index], err = g.mapGitspaceSettings(in[index])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (g gitspaceSettingsStore) mapToInternalGitspaceSettings(
	in *types.GitspaceSettings,
) (*gitspaceSettings, error) {
	var settingsBytes []byte
	var marshalErr error
	settingsBytes, marshalErr = json.Marshal(in.Settings)
	if marshalErr != nil {
		return nil, marshalErr
	}
	gitspaceSettingsEntity := &gitspaceSettings{
		ID:           in.ID,
		SpaceID:      in.SpaceID,
		SettingsData: settingsBytes,
		SettingsType: in.SettingsType,
		CriteriaKey:  in.CriteriaKey,
		Created:      in.Created,
		Updated:      in.Updated,
	}
	return gitspaceSettingsEntity, nil
}
