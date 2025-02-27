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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	infraProviderConfigIDColumn      = `ipconf_id`
	infraProviderConfigInsertColumns = `
		ipconf_uid,
		ipconf_display_name,
		ipconf_type,
		ipconf_space_id,
		ipconf_created,
		ipconf_updated,
		ipconf_metadata
	`
	infraProviderConfigSelectColumns = "ipconf_id," + infraProviderConfigInsertColumns
	infraProviderConfigTable         = `infra_provider_configs`
)

type infraProviderConfig struct {
	ID         int64                  `db:"ipconf_id"`
	Identifier string                 `db:"ipconf_uid"`
	Name       string                 `db:"ipconf_display_name"`
	Type       enum.InfraProviderType `db:"ipconf_type"`
	Metadata   []byte                 `db:"ipconf_metadata"`
	SpaceID    int64                  `db:"ipconf_space_id"`
	Created    int64                  `db:"ipconf_created"`
	Updated    int64                  `db:"ipconf_updated"`
}

var _ store.InfraProviderConfigStore = (*infraProviderConfigStore)(nil)

func NewInfraProviderConfigStore(db *sqlx.DB) store.InfraProviderConfigStore {
	return &infraProviderConfigStore{
		db: db,
	}
}

type infraProviderConfigStore struct {
	db *sqlx.DB
}

func (i infraProviderConfigStore) Update(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	dbinfraProviderConfig, err := i.mapToInternalInfraProviderConfig(infraProviderConfig)
	if err != nil {
		return err
	}
	stmt := database.Builder.
		Update(infraProviderConfigTable).
		Set("ipconf_display_name", dbinfraProviderConfig.Name).
		Set("ipconf_updated", dbinfraProviderConfig.Updated).
		Set("ipconf_metadata", dbinfraProviderConfig.Metadata).
		Where("ipconf_id = ?", infraProviderConfig.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, i.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "Failed to update infra provider config %s", infraProviderConfig.Identifier)
	}
	return nil
}

func (i infraProviderConfigStore) Find(ctx context.Context, id int64) (*types.InfraProviderConfig, error) {
	stmt := database.Builder.
		Select(infraProviderConfigSelectColumns).
		From(infraProviderConfigTable).
		Where(infraProviderConfigIDColumn+" = $1", id) //nolint:goconst
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderConfig)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider config %d", id)
	}
	return i.mapToInfraProviderConfig(dst)
}

func (i infraProviderConfigStore) FindByType(
	ctx context.Context,
	spaceID int64,
	infraProviderType enum.InfraProviderType,
) ([]*types.InfraProviderConfig, error) {
	stmt := database.Builder.
		Select(infraProviderConfigSelectColumns).
		From(infraProviderConfigTable).
		Where("ipconf_type = $1", infraProviderType). //nolint:goconst
		Where("ipconf_space_id = $2", spaceID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)
	dst := new([]*infraProviderConfig)
	if err := db.SelectContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list infraprovider configs of type %s for"+
			" space %d", infraProviderType, spaceID)
	}
	return i.mapToInfraProviderConfigs(*dst)
}

func (i infraProviderConfigStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.InfraProviderConfig, error) {
	stmt := database.Builder.
		Select(infraProviderConfigSelectColumns).
		From(infraProviderConfigTable).
		Where("ipconf_uid = $1", identifier). //nolint:goconst
		Where("ipconf_space_id = $2", spaceID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderConfig)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider config %s", identifier)
	}
	return i.mapToInfraProviderConfig(dst)
}

func (i infraProviderConfigStore) Create(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	dbinfraProviderConfig, err := i.mapToInternalInfraProviderConfig(infraProviderConfig)
	if err != nil {
		return err
	}
	stmt := database.Builder.
		Insert(infraProviderConfigTable).
		Columns(infraProviderConfigInsertColumns).
		Values(
			dbinfraProviderConfig.Identifier,
			dbinfraProviderConfig.Name,
			dbinfraProviderConfig.Type,
			dbinfraProviderConfig.SpaceID,
			dbinfraProviderConfig.Created,
			dbinfraProviderConfig.Updated,
			dbinfraProviderConfig.Metadata,
		).
		Suffix(ReturningClause + infraProviderConfigIDColumn)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, i.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&dbinfraProviderConfig.ID); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "infraprovider config create query failed for %s", dbinfraProviderConfig.Identifier)
	}
	return nil
}

func (i infraProviderConfigStore) mapToInfraProviderConfig(
	in *infraProviderConfig,
) (*types.InfraProviderConfig, error) {
	metadataMap := make(map[string]any)
	if len(in.Metadata) > 0 {
		marshalErr := json.Unmarshal(in.Metadata, &metadataMap)
		if marshalErr != nil {
			return nil, marshalErr
		}
	}
	infraProviderConfigEntity := &types.InfraProviderConfig{
		ID:         in.ID,
		Identifier: in.Identifier,
		Name:       in.Name,
		Type:       in.Type,
		Metadata:   metadataMap,
		SpaceID:    in.SpaceID,
		Created:    in.Created,
		Updated:    in.Updated,
	}
	return infraProviderConfigEntity, nil
}

func (i infraProviderConfigStore) mapToInfraProviderConfigs(
	in []*infraProviderConfig,
) ([]*types.InfraProviderConfig, error) {
	var err error
	res := make([]*types.InfraProviderConfig, len(in))
	for index := range in {
		res[index], err = i.mapToInfraProviderConfig(in[index])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (i infraProviderConfigStore) mapToInternalInfraProviderConfig(
	in *types.InfraProviderConfig,
) (*infraProviderConfig, error) {
	var jsonBytes []byte
	var marshalErr error
	if len(in.Metadata) > 0 {
		jsonBytes, marshalErr = json.Marshal(in.Metadata)
		if marshalErr != nil {
			return nil, marshalErr
		}
	}
	infraProviderConfigEntity := &infraProviderConfig{
		Identifier: in.Identifier,
		Name:       in.Name,
		Type:       in.Type,
		SpaceID:    in.SpaceID,
		Created:    in.Created,
		Updated:    in.Updated,
		Metadata:   jsonBytes,
	}
	return infraProviderConfigEntity, nil
}
