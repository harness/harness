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
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.SettingsStore = (*SettingsStore)(nil)

// NewSettingsStore returns a new SettingsStore.
func NewSettingsStore(db *sqlx.DB) *SettingsStore {
	return &SettingsStore{
		db: db,
	}
}

// SettingsStore implements store.SettingsStore backed by a relational database.
type SettingsStore struct {
	db *sqlx.DB
}

// setting is an internal representation used to store setting data in the database.
type setting struct {
	ID      int64           `db:"setting_id"`
	SpaceID null.Int        `db:"setting_space_id"`
	RepoID  null.Int        `db:"setting_repo_id"`
	Key     string          `db:"setting_key"`
	Value   json.RawMessage `db:"setting_value"`
}

const (
	settingsColumns = `
		 setting_id
		,setting_space_id
		,setting_repo_id
		,setting_key
		,setting_value`
)

func (s *SettingsStore) Find(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) (json.RawMessage, error) {
	stmt := database.Builder.
		Select(settingsColumns).
		From("settings").
		Where("LOWER(setting_key) = ?", strings.ToLower(key))

	switch scope {
	case enum.SettingsScopeSpace:
		stmt = stmt.Where("setting_space_id = ?", scopeID)
	case enum.SettingsScopeRepo:
		stmt = stmt.Where("setting_repo_id = ?", scopeID)
	case enum.SettingsScopeSystem:
		stmt = stmt.Where("setting_repo_id IS NULL AND setting_space_id IS NULL")
	default:
		return nil, fmt.Errorf("setting scope %q is not supported", scope)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &setting{}
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return dst.Value, nil
}

func (s *SettingsStore) FindMany(
	ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) (map[string]json.RawMessage, error) {
	if len(keys) == 0 {
		return map[string]json.RawMessage{}, nil
	}

	keysLower := make([]string, len(keys))
	for i, k := range keys {
		keysLower[i] = strings.ToLower(k)
	}

	stmt := database.Builder.
		Select(settingsColumns).
		From("settings").
		Where(squirrel.Eq{"LOWER(setting_key)": keysLower})

	switch scope {
	case enum.SettingsScopeSpace:
		stmt = stmt.Where("setting_space_id = ?", scopeID)
	case enum.SettingsScopeRepo:
		stmt = stmt.Where("setting_repo_id = ?", scopeID)
	case enum.SettingsScopeSystem:
		stmt = stmt.Where("setting_repo_id IS NULL AND setting_space_id IS NULL")
	default:
		return nil, fmt.Errorf("setting scope %q is not supported", scope)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*setting{}
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	out := map[string]json.RawMessage{}
	for _, d := range dst {
		out[d.Key] = d.Value
	}

	return out, nil
}

func (s *SettingsStore) Upsert(ctx context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
	value json.RawMessage,
) error {
	stmt := database.Builder.
		Insert("").
		Into("settings").
		Columns(
			"setting_space_id",
			"setting_repo_id",
			"setting_key",
			"setting_value",
		)

	switch scope {
	case enum.SettingsScopeSpace:
		stmt = stmt.Values(null.IntFrom(scopeID), null.Int{}, key, value)
		stmt = stmt.Suffix(`ON CONFLICT (setting_space_id, LOWER(setting_key)) WHERE setting_space_id IS NOT NULL DO`)
	case enum.SettingsScopeRepo:
		stmt = stmt.Values(null.Int{}, null.IntFrom(scopeID), key, value)
		stmt = stmt.Suffix(`ON CONFLICT (setting_repo_id, LOWER(setting_key)) WHERE setting_repo_id IS NOT NULL DO`)
	case enum.SettingsScopeSystem:
		stmt = stmt.Values(null.Int{}, null.Int{}, key, value)
		stmt = stmt.Suffix(`ON CONFLICT (LOWER(setting_key)) 
			WHERE setting_repo_id IS NULL AND setting_space_id IS NULL DO`)
	default:
		return fmt.Errorf("setting scope %q is not supported", scope)
	}

	stmt = stmt.Suffix(`
	UPDATE SET
		setting_value = EXCLUDED.setting_value
	WHERE
	`)
	if strings.HasPrefix(s.db.DriverName(), "sqlite") {
		stmt = stmt.Suffix(`settings.setting_value <> EXCLUDED.setting_value`)
	} else {
		stmt = stmt.Suffix(`settings.setting_value::text <> EXCLUDED.setting_value::text`)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Upsert query failed")
	}

	return nil
}
