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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.CDEGatewayStore = (*CDEGatewayStore)(nil)

const (
	cdeGatewayIDColumn      = `cgate_id`
	cdeGatewayInsertColumns = `
		cgate_name,
		cgate_group_name,
		cgate_space_id,
		cgate_infra_provider_config_id,
		cgate_region,
		cgate_zone,
		cgate_version,
		cgate_health,
		cgate_envoy_health,
		cgate_created,
		cgate_updated
	`
	cdeGatewaySelectColumns = cdeGatewayIDColumn + "," + cdeGatewayInsertColumns
	cdeGatewayTable         = `cde_gateways`
)

// NewCDEGatewayStore returns a new CDEGatewayStore.
func NewCDEGatewayStore(db *sqlx.DB) *CDEGatewayStore {
	return &CDEGatewayStore{
		db: db,
	}
}

// CDEGatewayStore implements store.CDEGatewayStore backed by a relational database.
type CDEGatewayStore struct {
	db *sqlx.DB
}

func (c *CDEGatewayStore) Upsert(ctx context.Context, in *types.CDEGateway) error {
	stmt := database.Builder.
		Insert(cdeGatewayTable).
		Columns(cdeGatewayInsertColumns).
		Values(
			in.Name,
			in.GroupName,
			in.SpaceID,
			in.InfraProviderConfigID,
			in.Region,
			in.Zone,
			in.Version,
			in.Health,
			in.EnvoyHealth,
			in.Created,
			in.Updated).
		Suffix(`
ON CONFLICT (cgate_space_id, cgate_infra_provider_config_id, cgate_region, cgate_group_name, cgate_name)
DO UPDATE 
SET 
	cgate_health = EXCLUDED.cgate_health,
	cgate_envoy_health = EXCLUDED.cgate_envoy_health,
	cgate_updated = EXCLUDED.cgate_updated,
	cgate_zone = EXCLUDED.cgate_zone,
	cgate_version = EXCLUDED.cgate_version`)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, c.db)
	if _, err = db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "cde gateway upsert create query failed for %s", in.Name)
	}
	return nil
}

func (c *CDEGatewayStore) List(ctx context.Context, filter *types.CDEGatewayFilter) ([]*types.CDEGateway, error) {
	stmt := database.Builder.
		Select(cdeGatewaySelectColumns).
		From(cdeGatewayTable)

	if filter != nil && len(filter.InfraProviderConfigIDs) > 0 {
		stmt = stmt.Where(squirrel.Eq{"cgate_infra_provider_config_id": filter.InfraProviderConfigIDs})
	}

	if filter != nil && filter.Health == types.GatewayHealthHealthy {
		stmt = stmt.Where(squirrel.Eq{"cgate_health": filter.Health}).
			Where(squirrel.Eq{"cgate_envoy_health": filter.Health}).
			Where(squirrel.Gt{"cgate_updated": time.Now().Add(
				-time.Duration(filter.HealthReportValidityInMins) * time.Minute).UnixMilli()})
	}

	if filter != nil && filter.Health == types.GatewayHealthUnhealthy {
		stmt = stmt.Where(
			squirrel.Or{
				squirrel.LtOrEq{"cgate_updated": time.Now().Add(
					time.Minute * -time.Duration(filter.HealthReportValidityInMins)).UnixMilli()},
				squirrel.Eq{"cgate_envoy_health": filter.Health},
			},
		)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, c.db)
	dst := new([]*cdeGateway)
	if err := db.SelectContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list cde gatways")
	}

	return entitiesToDTOs(*dst), nil
}

type cdeGateway struct {
	ID                    int64  `db:"cgate_id"`
	Name                  string `db:"cgate_name"`
	GroupName             string `db:"cgate_group_name"`
	SpaceID               int64  `db:"cgate_space_id"`
	InfraProviderConfigID int64  `db:"cgate_infra_provider_config_id"`
	Region                string `db:"cgate_region"`
	Zone                  string `db:"cgate_zone"`
	Version               string `db:"cgate_version"`
	Health                string `db:"cgate_health"`
	EnvoyHealth           string `db:"cgate_envoy_health"`
	Created               int64  `db:"cgate_created"`
	Updated               int64  `db:"cgate_updated"`
}

func entitiesToDTOs(entities []*cdeGateway) []*types.CDEGateway {
	var dtos []*types.CDEGateway
	for _, entity := range entities {
		dtos = append(dtos, entityToDTO(*entity))
	}
	return dtos
}

func entityToDTO(entity cdeGateway) *types.CDEGateway {
	dto := &types.CDEGateway{}
	dto.Name = entity.Name
	dto.GroupName = entity.GroupName
	dto.SpaceID = entity.SpaceID
	dto.InfraProviderConfigID = entity.InfraProviderConfigID
	dto.Region = entity.Region
	dto.Zone = entity.Zone
	dto.Version = entity.Version
	dto.Health = entity.Health
	dto.EnvoyHealth = entity.EnvoyHealth
	dto.Created = entity.Created
	dto.Updated = entity.Updated
	return dto
}
