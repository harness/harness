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

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	infraProviderResourceIDColumn      = `ipreso_id`
	infraProviderResourceInsertColumns = `
		ipreso_uid,
		ipreso_display_name,
		ipreso_infra_provider_config_id,
		ipreso_type,
		ipreso_space_id,
		ipreso_created,
		ipreso_updated,
		ipreso_cpu,
		ipreso_memory,
		ipreso_disk,
		ipreso_network,
		ipreso_region,
		ipreso_opentofu_params,
		ipreso_gateway_host,
		ipreso_gateway_port,
		ipreso_infra_provider_template_id
	`
	infraProviderResourceSelectColumns = "ipreso_id," + infraProviderResourceInsertColumns
	infraProviderResourceTable         = `infra_provider_resources`
)

type infraProviderResource struct {
	ID                    int64                  `db:"ipreso_id"`
	Identifier            string                 `db:"ipreso_uid"`
	Name                  string                 `db:"ipreso_display_name"`
	InfraProviderConfigID int64                  `db:"ipreso_infra_provider_config_id"`
	InfraProviderType     enum.InfraProviderType `db:"ipreso_type"`
	SpaceID               int64                  `db:"ipreso_space_id"`
	CPU                   null.String            `db:"ipreso_cpu"`
	Memory                null.String            `db:"ipreso_memory"`
	Disk                  null.String            `db:"ipreso_disk"`
	Network               null.String            `db:"ipreso_network"`
	Region                string                 `db:"ipreso_region"` // need list maybe
	OpenTofuParams        []byte                 `db:"ipreso_opentofu_params"`
	GatewayHost           null.String            `db:"ipreso_gateway_host"`
	GatewayPort           null.String            `db:"ipreso_gateway_port"`
	TemplateID            null.Int               `db:"ipreso_infra_provider_template_id"`
	Created               int64                  `db:"ipreso_created"`
	Updated               int64                  `db:"ipreso_updated"`
}

var _ store.InfraProviderResourceStore = (*infraProviderResourceStore)(nil)

// NewGitspaceConfigStore returns a new GitspaceConfigStore.
func NewInfraProviderResourceStore(db *sqlx.DB) store.InfraProviderResourceStore {
	return &infraProviderResourceStore{
		db: db,
	}
}

type infraProviderResourceStore struct {
	db *sqlx.DB
}

func (s infraProviderResourceStore) List(ctx context.Context, infraProviderConfigID int64,
	_ types.ListQueryFilter) ([]*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where("ipreso_infra_provider_config_id = $1", infraProviderConfigID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	dst := new([]infraProviderResource)
	if err := db.SelectContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return s.mapToInfraProviderResources(ctx, *dst)
}

func (s infraProviderResourceStore) mapToInfraProviderResource(_ context.Context,
	in *infraProviderResource) (*types.InfraProviderResource, error) {
	openTofuParamsMap := make(map[string]string)
	marshalErr := json.Unmarshal(in.OpenTofuParams, &openTofuParamsMap)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return &types.InfraProviderResource{
		Identifier:            in.Identifier,
		InfraProviderConfigID: in.InfraProviderConfigID,
		ID:                    in.ID,
		InfraProviderType:     in.InfraProviderType,
		Name:                  in.Name,
		SpaceID:               in.SpaceID,
		CPU:                   in.CPU.Ptr(),
		Memory:                in.Memory.Ptr(),
		Disk:                  in.Disk.Ptr(),
		Network:               in.Network.Ptr(),
		Region:                in.Region,
		Metadata:              openTofuParamsMap,
		GatewayHost:           in.GatewayHost.Ptr(),
		GatewayPort:           in.GatewayPort.Ptr(),
		TemplateID:            in.TemplateID.Ptr(),
		Created:               in.Created,
		Updated:               in.Updated,
	}, nil
}

func (s infraProviderResourceStore) Find(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where(infraProviderResourceIDColumn+" = $1", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderResource)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProviderResource")
	}
	return s.mapToInfraProviderResource(ctx, dst)
}

func (s infraProviderResourceStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where("ipreso_uid = $1", identifier).
		Where("ipreso_space_id = $2", spaceID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderResource)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProviderResource")
	}
	return s.mapToInfraProviderResource(ctx, dst)
}

func (s infraProviderResourceStore) Create(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
) error {
	jsonBytes, marshalErr := json.Marshal(infraProviderResource.Metadata)
	if marshalErr != nil {
		return marshalErr
	}
	stmt := database.Builder.
		Insert(infraProviderResourceTable).
		Columns(infraProviderResourceInsertColumns).
		Values(
			infraProviderResource.Identifier,
			infraProviderResource.Name,
			infraProviderResource.InfraProviderConfigID,
			infraProviderResource.InfraProviderType,
			infraProviderResource.SpaceID,
			infraProviderResource.Created,
			infraProviderResource.Updated,
			infraProviderResource.CPU,
			infraProviderResource.Memory,
			infraProviderResource.Disk,
			infraProviderResource.Network,
			infraProviderResource.Region,
			jsonBytes,
			infraProviderResource.GatewayHost,
			infraProviderResource.GatewayPort,
			infraProviderResource.TemplateID,
		).
		Suffix(ReturningClause + infraProviderResourceIDColumn)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&infraProviderResource.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "infra provider resource query failed")
	}
	return nil
}

func (s infraProviderResourceStore) DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error {
	stmt := database.Builder.
		Delete(infraProviderResourceTable).
		Where("ipreso_uid = $1", identifier).
		Where("ipreso_space_id = $2", spaceID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete infra provider resource")
	}
	return nil
}

func (s infraProviderResourceStore) mapToInfraProviderResources(ctx context.Context,
	resources []infraProviderResource) ([]*types.InfraProviderResource, error) {
	var err error
	res := make([]*types.InfraProviderResource, len(resources))
	for i := range resources {
		res[i], err = s.mapToInfraProviderResource(ctx, &resources[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
