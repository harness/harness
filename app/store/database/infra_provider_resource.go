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
	"time"

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
		ipreso_metadata,
		ipreso_is_deleted,
		ipreso_deleted
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
	Metadata              []byte                 `db:"ipreso_metadata"`
	Created               int64                  `db:"ipreso_created"`
	Updated               int64                  `db:"ipreso_updated"`
	IsDeleted             bool                   `db:"ipreso_is_deleted"`
	Deleted               null.Int               `db:"ipreso_deleted"`
}

var _ store.InfraProviderResourceStore = (*infraProviderResourceStore)(nil)

// NewInfraProviderResourceStore returns a new InfraProviderResourceStore.
func NewInfraProviderResourceStore(db *sqlx.DB) store.InfraProviderResourceStore {
	return &infraProviderResourceStore{
		db: db,
	}
}

type infraProviderResourceStore struct {
	db *sqlx.DB
}

func (s infraProviderResourceStore) List(
	ctx context.Context,
	infraProviderConfigID int64,
	_ types.ListQueryFilter,
) ([]*types.InfraProviderResource, error) {
	subQuery := squirrel.Select("MAX(ipreso_created)").
		From(infraProviderResourceTable).
		Where("ipreso_infra_provider_config_id = $1", infraProviderConfigID).
		Where("ipreso_is_deleted = false").
		GroupBy("ipreso_uid")

	stmt := squirrel.Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where("ipreso_infra_provider_config_id = $2", infraProviderConfigID).
		Where("ipreso_is_deleted = false").
		Where(squirrel.Expr("ipreso_created IN (?)", subQuery)).
		OrderBy("ipreso_uid", "ipreso_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	dst := new([]infraProviderResource)
	if err := db.SelectContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list infraprovider resources for config %d",
			infraProviderConfigID)
	}
	return mapToInfraProviderResources(*dst)
}

func (s infraProviderResourceStore) Find(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where(infraProviderResourceIDColumn+" = $1", id).
		Where("ipreso_is_deleted = false")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderResource)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider resource %d", id)
	}
	return mapToInfraProviderResource(dst)
}

func (s infraProviderResourceStore) FindByConfigAndIdentifier(
	ctx context.Context,
	spaceID int64,
	infraProviderConfigID int64,
	identifier string,
) (*types.InfraProviderResource, error) {
	stmt :=
		database.Builder.
			Select(infraProviderResourceSelectColumns).
			From(infraProviderResourceTable).
			OrderBy("ipreso_created DESC").
			Limit(1).
			Where("ipreso_uid = ?", identifier).
			Where("ipreso_space_id = ?", spaceID).
			Where("ipreso_infra_provider_config_id = ?", infraProviderConfigID).
			Where("ipreso_is_deleted = false")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderResource)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider resource %s", identifier)
	}
	return mapToInfraProviderResource(dst)
}

func (s infraProviderResourceStore) Create(
	ctx context.Context,
	infraProviderResource *types.InfraProviderResource,
) error {
	metadata, marshalErr := json.Marshal(infraProviderResource.Metadata)
	if marshalErr != nil {
		return marshalErr
	}
	stmt := database.Builder.
		Insert(infraProviderResourceTable).
		Columns(infraProviderResourceInsertColumns).
		Values(
			infraProviderResource.UID,
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
			metadata,
			infraProviderResource.IsDeleted,
			infraProviderResource.Deleted,
		).
		Suffix(ReturningClause + infraProviderResourceIDColumn)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&infraProviderResource.ID); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "infra provider resource create failed %s", infraProviderResource.UID)
	}
	return nil
}

func mapToInfraProviderResource(in *infraProviderResource) (*types.InfraProviderResource, error) {
	metadataParamsMap := make(map[string]string)
	marshalErr := json.Unmarshal(in.Metadata, &metadataParamsMap)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return &types.InfraProviderResource{
		UID:                   in.Identifier,
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
		Metadata:              metadataParamsMap,
		Created:               in.Created,
		Updated:               in.Updated,
		IsDeleted:             in.IsDeleted,
		Deleted:               in.Deleted.Ptr(),
	}, nil
}

func mapToInfraProviderResources(resources []infraProviderResource) ([]*types.InfraProviderResource, error) {
	var err error
	res := make([]*types.InfraProviderResource, len(resources))
	for i := range resources {
		res[i], err = mapToInfraProviderResource(&resources[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

var _ store.InfraProviderResourceView = (*InfraProviderResourceView)(nil)

// NewInfraProviderResourceView returns a new InfraProviderResourceView.
// It's used by the infraprovider resource cache.
func NewInfraProviderResourceView(db *sqlx.DB, spaceStore store.SpaceStore) *InfraProviderResourceView {
	return &InfraProviderResourceView{
		db:         db,
		spaceStore: spaceStore,
	}
}

type InfraProviderResourceView struct {
	db         *sqlx.DB
	spaceStore store.SpaceStore
}

func (i InfraProviderResourceView) Find(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where(infraProviderResourceIDColumn+" = $1", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(infraProviderResource)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider providerResource %d", id)
	}
	providerResource, err := mapToInfraProviderResource(dst)
	if err != nil {
		return nil, err
	}
	providerConfig, err := i.findInfraProviderConfig(ctx, providerResource.InfraProviderConfigID)
	if err == nil && providerConfig != nil {
		providerResource.InfraProviderConfigIdentifier = providerConfig.Identifier
	}
	resourceSpace, err := i.spaceStore.Find(ctx, providerResource.SpaceID)
	if err == nil {
		providerResource.SpacePath = resourceSpace.Path
	}
	return providerResource, err
}

func (i InfraProviderResourceView) findInfraProviderConfig(
	ctx context.Context,
	id int64,
) (*infraProviderConfig, error) {
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
	return dst, nil
}

func (i InfraProviderResourceView) FindMany(ctx context.Context, ids []int64) ([]*types.InfraProviderResource, error) {
	stmt := database.Builder.
		Select(infraProviderResourceSelectColumns).
		From(infraProviderResourceTable).
		Where(squirrel.Eq{infraProviderResourceIDColumn: ids})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new([]infraProviderResource)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraprovider resources")
	}
	return mapToInfraProviderResources(*dst)
}

func (s infraProviderResourceStore) Delete(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	stmt := database.Builder.
		Update(infraProviderResourceTable).
		Set("ipreso_updated", now).
		Set("ipreso_deleted", now).
		Set("ipreso_is_deleted", true).
		Where("ipreso_id = $4", id)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "Failed to update infraprovider resource %d", id)
	}
	return nil
}
