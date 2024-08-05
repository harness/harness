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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	infraProviderTemplateIDColumn = `iptemp_id`
	infraProviderTemplateColumns  = `
		iptemp_uid,
    	iptemp_infra_provider_config_id,
    	iptemp_description,
    	iptemp_space_id,
    	iptemp_data,
    	iptemp_created,
    	iptemp_updated,
    	iptemp_version	
	`
	infraProviderTemplateSelectColumns = infraProviderTemplateIDColumn + `,
    	` + infraProviderTemplateColumns
	infraProviderTemplateTable = `infra_provider_templates`
)

var _ store.InfraProviderTemplateStore = (*infraProviderTemplateStore)(nil)

type infraProviderTemplateStore struct {
	db *sqlx.DB
}

type infraProviderTemplate struct {
	ID                    int64  `db:"iptemp_id"`
	Identifier            string `db:"iptemp_uid"`
	InfraProviderConfigID int64  `db:"iptemp_infra_provider_config_id"`
	Description           string `db:"iptemp_description"`
	SpaceID               int64  `db:"iptemp_space_id"`
	Data                  string `db:"iptemp_data"`
	Created               int64  `db:"iptemp_created"`
	Updated               int64  `db:"iptemp_updated"`
	Version               int64  `db:"iptemp_version"`
}

func NewInfraProviderTemplateStore(db *sqlx.DB) store.InfraProviderTemplateStore {
	return &infraProviderTemplateStore{
		db: db,
	}
}

func (i infraProviderTemplateStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.InfraProviderTemplate, error) {
	stmt := database.Builder.
		Select(infraProviderTemplateSelectColumns).
		From(infraProviderTemplateTable).
		Where("iptemp_uid = $1", identifier).
		Where("iptemp_space_id = $2", spaceID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	infraProviderTemplateEntity := new(infraProviderTemplate)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, infraProviderTemplateEntity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProviderTemplate")
	}
	return infraProviderTemplateEntity.mapToDTO(), nil
}

func (i infraProviderTemplateStore) Find(
	ctx context.Context,
	id int64,
) (*types.InfraProviderTemplate,
	error) {
	stmt := database.Builder.
		Select(infraProviderTemplateSelectColumns).
		From(infraProviderTemplateTable).
		Where(infraProviderTemplateIDColumn+" = $1", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	infraProviderTemplateEntity := new(infraProviderTemplate)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, infraProviderTemplateEntity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProviderTemplate")
	}
	return infraProviderTemplateEntity.mapToDTO(), nil
}

func (i infraProviderTemplateStore) Create(
	ctx context.Context,
	infraProviderTemplate *types.InfraProviderTemplate,
) error {
	stmt := database.Builder.
		Insert(infraProviderTemplateTable).
		Columns(infraProviderTemplateColumns).
		Values(infraProviderTemplate.Identifier,
			infraProviderTemplate.InfraProviderConfigID,
			infraProviderTemplate.Description,
			infraProviderTemplate.SpaceID,
			infraProviderTemplate.Data,
			infraProviderTemplate.Created,
			infraProviderTemplate.Updated,
			infraProviderTemplate.Version,
		).
		Suffix(ReturningClause + infraProviderTemplateIDColumn)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, i.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&infraProviderTemplate.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "infraProviderTemplate query failed")
	}
	return nil
}

func (i infraProviderTemplateStore) Delete(ctx context.Context, id int64) error {
	stmt := database.Builder.
		Delete(infraProviderTemplateTable).
		Where(infraProviderTemplateIDColumn+" = $1", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, i.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete infra provider template")
	}
	return nil
}

func (entity infraProviderTemplate) mapToDTO() *types.InfraProviderTemplate {
	return &types.InfraProviderTemplate{
		ID:                    entity.ID,
		Identifier:            entity.Identifier,
		InfraProviderConfigID: entity.InfraProviderConfigID,
		Description:           entity.Description,
		Data:                  entity.Data,
		Version:               entity.Version,
		SpaceID:               entity.SpaceID,
		Created:               entity.Created,
		Updated:               entity.Updated,
	}
}
