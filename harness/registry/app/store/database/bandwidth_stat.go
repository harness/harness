//  Copyright 2023 Harness, Inc.
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
	"database/sql"
	"errors"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type BandwidthStatDao struct {
	db *sqlx.DB
}

func NewBandwidthStatDao(db *sqlx.DB) store.BandwidthStatRepository {
	return &BandwidthStatDao{
		db: db,
	}
}

type bandwidthStatDB struct {
	ID        int64               `db:"bandwidth_stat_id"`
	ImageID   int64               `db:"bandwidth_stat_image_id"`
	Timestamp int64               `db:"bandwidth_stat_timestamp"`
	Type      types.BandwidthType `db:"bandwidth_stat_type"`
	Bytes     int64               `db:"bandwidth_stat_bytes"`
	CreatedAt int64               `db:"bandwidth_stat_created_at"`
	UpdatedAt int64               `db:"bandwidth_stat_updated_at"`
	CreatedBy int64               `db:"bandwidth_stat_created_by"`
	UpdatedBy int64               `db:"bandwidth_stat_updated_by"`
}

func (b BandwidthStatDao) Create(ctx context.Context, bandwidthStat *types.BandwidthStat) error {
	const sqlQuery = `
		INSERT INTO bandwidth_stats ( 
		         bandwidth_stat_image_id
				,bandwidth_stat_timestamp
				,bandwidth_stat_type
				,bandwidth_stat_bytes
				,bandwidth_stat_created_at
				,bandwidth_stat_updated_at
				,bandwidth_stat_created_by
				,bandwidth_stat_updated_by		
		    ) VALUES (
						 :bandwidth_stat_image_id
						,:bandwidth_stat_timestamp
						,:bandwidth_stat_type
						,:bandwidth_stat_bytes
						,:bandwidth_stat_created_at
						,:bandwidth_stat_updated_at
				        ,:bandwidth_stat_created_by
				        ,:bandwidth_stat_updated_by							
		    ) 		   
        RETURNING bandwidth_stat_id`

	db := dbtx.GetAccessor(ctx, b.db)
	query, arg, err := db.BindNamed(sqlQuery, b.mapToInternalBandwidthStat(ctx, bandwidthStat))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind bandwidth stat object")
	}

	if err = db.QueryRowContext(ctx, query,
		arg...).Scan(&bandwidthStat.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (b BandwidthStatDao) mapToInternalBandwidthStat(ctx context.Context,
	in *types.BandwidthStat) *bandwidthStatDB {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}

	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()

	return &bandwidthStatDB{
		ID:        in.ID,
		ImageID:   in.ImageID,
		Timestamp: time.Now().UnixMilli(),
		Type:      in.Type,
		Bytes:     in.Bytes,
		CreatedAt: in.CreatedAt.UnixMilli(),
		UpdatedAt: in.UpdatedAt.UnixMilli(),
		CreatedBy: in.CreatedBy,
		UpdatedBy: session.Principal.ID,
	}
}
