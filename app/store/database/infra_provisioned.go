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
	"github.com/pkg/errors"
)

const (
	infraProvisionedIDColumn = `iprov_id`
	infraProvisionedColumns  = `
		iprov_gitspace_id,
		iprov_type,
		iprov_infra_provider_resource_id,
		iprov_space_id,
		iprov_created,
		iprov_updated,
		iprov_response_metadata,
		iprov_opentofu_params,
		iprov_infra_status,
		iprov_server_host_ip,
		iprov_server_host_port,	
		iprov_proxy_host,
		iprov_proxy_port
	`
	infraProvisionedSelectColumns = infraProvisionedIDColumn + `,
		` + infraProvisionedColumns
	infraProvisionedTable = `infra_provisioned`
)

var _ store.InfraProvisionedStore = (*infraProvisionedStore)(nil)

type infraProvisionedStore struct {
	db *sqlx.DB
}

type infraProvisioned struct {
	ID                      int64                  `db:"iprov_id"`
	GitspaceInstanceID      int64                  `db:"iprov_gitspace_id"`
	InfraProviderType       enum.InfraProviderType `db:"iprov_type"`
	InfraProviderResourceID int64                  `db:"iprov_infra_provider_resource_id"`
	SpaceID                 int64                  `db:"iprov_space_id"`
	Created                 int64                  `db:"iprov_created"`
	Updated                 int64                  `db:"iprov_updated"`
	ResponseMetadata        *string                `db:"iprov_response_metadata"`
	InputParams             string                 `db:"iprov_opentofu_params"`
	InfraStatus             enum.InfraStatus       `db:"iprov_infra_status"`
	ServerHostIP            string                 `db:"iprov_server_host_ip"`
	ServerHostPort          string                 `db:"iprov_server_host_port"`
	ProxyHost               string                 `db:"iprov_proxy_host"`
	ProxyPort               int32                  `db:"iprov_proxy_port"`
}

type infraProvisionedGatewayView struct {
	GitspaceInstanceIdentifier string `db:"iprov_gitspace_uid"`
	SpaceID                    int64  `db:"iprov_space_id"`
	ServerHostIP               string `db:"iprov_server_host_ip"`
	ServerHostPort             string `db:"iprov_server_host_port"`
}

func NewInfraProvisionedStore(db *sqlx.DB) store.InfraProvisionedStore {
	return &infraProvisionedStore{
		db: db,
	}
}

func (i infraProvisionedStore) Find(ctx context.Context, id int64) (*types.InfraProvisioned, error) {
	stmt := database.Builder.
		Select(infraProvisionedSelectColumns).
		From(infraProvisionedTable).
		Where(infraProvisionedIDColumn+" = ?", id) // nolint:goconst

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	entity := new(infraProvisioned)
	db := dbtx.GetAccessor(ctx, i.db)
	err = db.GetContext(ctx, entity, sql, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProvisioned")
	}

	return entity.toDTO(), nil
}

func (i infraProvisionedStore) FindAllLatestByGateway(
	ctx context.Context,
	gatewayHost string,
) ([]*types.InfraProvisionedGatewayView, error) {
	stmt := database.Builder.
		Select("gits_uid as iprov_gitspace_uid, iprov_space_id, iprov_server_host_ip, iprov_server_host_port").
		From(infraProvisionedTable).
		Join(fmt.Sprintf("%s ON iprov_infra_provider_resource_id = ipreso_id", infraProviderResourceTable)).
		Join(fmt.Sprintf("%s ON iprov_gitspace_id = gits_id", gitspaceInstanceTable)).
		Where("ipreso_gateway_host = ?", gatewayHost).
		Where("iprov_infra_status = ?", enum.InfraStatusProvisioned).
		OrderBy("iprov_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	var entities []*infraProvisionedGatewayView
	db := dbtx.GetAccessor(ctx, i.db)
	err = db.SelectContext(ctx, &entities, sql, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProvisioned")
	}

	var result = make([]*types.InfraProvisionedGatewayView, len(entities))
	for index, entity := range entities {
		result[index] = &types.InfraProvisionedGatewayView{
			GitspaceInstanceIdentifier: entity.GitspaceInstanceIdentifier,
			SpaceID:                    entity.SpaceID,
			ServerHostIP:               entity.ServerHostIP,
			ServerHostPort:             entity.ServerHostPort,
		}
	}

	return result, nil
}

func (i infraProvisionedStore) FindLatestByGitspaceInstanceID(
	ctx context.Context,
	spaceID int64,
	gitspaceInstanceID int64,
) (*types.InfraProvisioned, error) {
	stmt := database.Builder.
		Select(infraProvisionedSelectColumns).
		From(infraProvisionedTable).
		Where("iprov_gitspace_id = ?", gitspaceInstanceID).
		Where("iprov_space_id = ?", spaceID).
		OrderBy("iprov_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	entity := new(infraProvisioned)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, entity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProvisioned")
	}

	return entity.toDTO(), nil
}

func (i infraProvisionedStore) FindLatestByGitspaceInstanceIdentifier(
	ctx context.Context,
	spaceID int64,
	gitspaceInstanceIdentifier string,
) (*types.InfraProvisioned, error) {
	stmt := database.Builder.
		Select(infraProvisionedSelectColumns).
		From(infraProvisionedTable).
		Join(fmt.Sprintf("%s ON iprov_gitspace_id = gits_id", gitspaceInstanceTable)).
		Where("gits_uid = ?", gitspaceInstanceIdentifier).
		Where("iprov_space_id = ?", spaceID).
		OrderBy("iprov_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	entity := new(infraProvisioned)
	db := dbtx.GetAccessor(ctx, i.db)
	if err := db.GetContext(ctx, entity, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find infraProvisioned")
	}

	return entity.toDTO(), nil
}

func (i infraProvisionedStore) Create(ctx context.Context, infraProvisioned *types.InfraProvisioned) error {
	stmt := database.Builder.
		Insert(infraProvisionedTable).
		Columns(infraProvisionedColumns).
		Values(
			infraProvisioned.GitspaceInstanceID,
			infraProvisioned.InfraProviderType,
			infraProvisioned.InfraProviderResourceID,
			infraProvisioned.SpaceID,
			infraProvisioned.Created,
			infraProvisioned.Updated,
			infraProvisioned.ResponseMetadata,
			infraProvisioned.InputParams,
			infraProvisioned.InfraStatus,
			infraProvisioned.ServerHostIP,
			infraProvisioned.ServerHostPort,
			infraProvisioned.ProxyHost,
			infraProvisioned.ProxyPort,
		).
		Suffix(ReturningClause + infraProvisionedIDColumn)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&infraProvisioned.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "infraProvisioned query failed")
	}

	return nil
}

func (i infraProvisionedStore) Delete(ctx context.Context, id int64) error {
	stmt := database.Builder.
		Delete(infraProvisionedTable).
		Where(infraProvisionedIDColumn+" = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete infra provisioned")
	}

	return nil
}

func (i infraProvisionedStore) Update(ctx context.Context, infraProvisioned *types.InfraProvisioned) error {
	stmt := database.Builder.
		Update(infraProvisionedTable).
		Set("iprov_response_metadata", infraProvisioned.ResponseMetadata).
		Set("iprov_infra_status", infraProvisioned.InfraStatus).
		Set("iprov_server_host_ip", infraProvisioned.ServerHostIP).
		Set("iprov_server_host_port", infraProvisioned.ServerHostPort).
		Set("iprov_opentofu_params", infraProvisioned.InputParams).
		Set("iprov_updated", infraProvisioned.Updated).
		Set("iprov_proxy_host", infraProvisioned.ProxyHost).
		Set("iprov_proxy_port", infraProvisioned.ProxyPort).
		Where(infraProvisionedIDColumn+" = ?", infraProvisioned.ID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update infra provisioned")
	}

	return nil
}

func (entity infraProvisioned) toDTO() *types.InfraProvisioned {
	return &types.InfraProvisioned{
		ID:                      entity.ID,
		GitspaceInstanceID:      entity.GitspaceInstanceID,
		InfraProviderType:       entity.InfraProviderType,
		InfraProviderResourceID: entity.InfraProviderResourceID,
		SpaceID:                 entity.SpaceID,
		Created:                 entity.Created,
		Updated:                 entity.Updated,
		ResponseMetadata:        entity.ResponseMetadata,
		InputParams:             entity.InputParams,
		InfraStatus:             entity.InfraStatus,
		ServerHostIP:            entity.ServerHostIP,
		ServerHostPort:          entity.ServerHostPort,
		ProxyHost:               entity.ProxyHost,
		ProxyPort:               entity.ProxyPort,
	}
}
