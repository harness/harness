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
	"errors"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.UsageMetricStore = (*UsageMetricsStore)(nil)

type usageMetric struct {
	RootSpaceID     int64 `db:"usage_metric_space_id"`
	Date            int64 `db:"usage_metric_date"`
	Created         int64 `db:"usage_metric_created"`
	Updated         int64 `db:"usage_metric_updated"`
	BandwidthOut    int64 `db:"usage_metric_bandwidth_out"`
	BandwidthIn     int64 `db:"usage_metric_bandwidth_in"`
	StorageTotal    int64 `db:"usage_metric_storage_total"`
	LFSStorageTotal int64 `db:"usage_metric_lfs_storage_total"`
	Pushes          int64 `db:"usage_metric_pushes"`
	Version         int64 `db:"usage_metric_version"`
}

// NewUsageMetricsStore returns a new UsageMetricsStore.
func NewUsageMetricsStore(db *sqlx.DB) *UsageMetricsStore {
	return &UsageMetricsStore{
		db: db,
	}
}

// UsageMetricsStore implements store.UsageMetrics backed by a relational database.
type UsageMetricsStore struct {
	db *sqlx.DB
}

func (s *UsageMetricsStore) getVersion(
	ctx context.Context,
	rootSpaceID int64,
	date int64,
) int64 {
	const sqlQuery = `
		SELECT
		    usage_metric_version
		FROM usage_metrics
		WHERE usage_metric_space_id = $1 AND usage_metric_date = $2
	`
	var version int64
	err := s.db.QueryRowContext(ctx, sqlQuery, rootSpaceID, date).Scan(&version)
	if err != nil {
		return 0
	}
	return version
}

func (s *UsageMetricsStore) Upsert(ctx context.Context, in *types.UsageMetric) error {
	sqlQuery := `
		INSERT INTO usage_metrics (
        	usage_metric_space_id
			,usage_metric_date
			,usage_metric_created
			,usage_metric_updated
			,usage_metric_bandwidth_out
			,usage_metric_bandwidth_in
			,usage_metric_storage_total
			,usage_metric_lfs_storage_total
			,usage_metric_pushes
			,usage_metric_version
		) VALUES (
			:usage_metric_space_id
		    ,:usage_metric_date
		    ,:usage_metric_created
		    ,:usage_metric_updated
		    ,:usage_metric_bandwidth_out
		    ,:usage_metric_bandwidth_in
		    ,:usage_metric_storage_total
		    ,:usage_metric_lfs_storage_total
		    ,:usage_metric_pushes
		    ,:usage_metric_version
		)
		ON CONFLICT (usage_metric_space_id, usage_metric_date)
		DO UPDATE
			SET
			    usage_metric_version = EXCLUDED.usage_metric_version
		        ,usage_metric_updated = EXCLUDED.usage_metric_updated
		        ,usage_metric_bandwidth_out = usage_metrics.usage_metric_bandwidth_out + EXCLUDED.usage_metric_bandwidth_out
				,usage_metric_bandwidth_in = usage_metrics.usage_metric_bandwidth_in + EXCLUDED.usage_metric_bandwidth_in
		        ,usage_metric_pushes = usage_metrics.usage_metric_pushes + EXCLUDED.usage_metric_pushes
`
	if in.StorageTotal > 0 {
		sqlQuery += `
				,usage_metric_storage_total = EXCLUDED.usage_metric_storage_total`
	}

	if in.LFSStorageTotal > 0 {
		sqlQuery += `
				,usage_metric_lfs_storage_total = EXCLUDED.usage_metric_lfs_storage_total`
	}

	sqlQuery += `
			WHERE usage_metrics.usage_metric_version = EXCLUDED.usage_metric_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)
	now := time.Now()
	today := s.Date(now)
	if !in.Date.IsZero() {
		today = s.Date(in.Date)
	}
	query, args, err := db.BindNamed(sqlQuery, usageMetric{
		RootSpaceID:     in.RootSpaceID,
		Date:            today,
		Created:         now.UnixMilli(),
		Updated:         now.UnixMilli(),
		BandwidthOut:    in.BandwidthOut,
		BandwidthIn:     in.BandwidthIn,
		StorageTotal:    in.StorageTotal,
		LFSStorageTotal: in.LFSStorageTotal,
		Pushes:          in.Pushes,
		Version:         s.getVersion(ctx, in.RootSpaceID, today) + 1,
	})
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to bind query")
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to upsert usage_metric")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to fetch number of rows affected")
	}
	if n == 0 {
		return gitness_store.ErrVersionConflict
	}
	return nil
}

// UpsertOptimistic upsert the usage metric details using the optimistic locking mechanism.
func (s *UsageMetricsStore) UpsertOptimistic(
	ctx context.Context,
	in *types.UsageMetric,
) error {
	for {
		err := s.Upsert(ctx, in)
		if err == nil {
			return nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return err
		}
	}
}

func (s *UsageMetricsStore) GetMetrics(
	ctx context.Context,
	rootSpaceID int64,
	start int64,
	end int64,
) (*types.UsageMetric, error) {
	const sqlQuery = `
	SELECT
		COALESCE(SUM(usage_metric_bandwidth_out), 0) AS usage_metric_bandwidth_out,
		COALESCE(SUM(usage_metric_bandwidth_in), 0) AS usage_metric_bandwidth_in,
		COALESCE(AVG(usage_metric_storage_total), 0) AS usage_metric_storage_total,
		COALESCE(AVG(usage_metric_lfs_storage_total), 0) AS usage_metric_lfs_storage_total,
		COALESCE(SUM(usage_metric_pushes), 0) AS usage_metric_pushes
	FROM usage_metrics
	WHERE
	    usage_metric_space_id = $1 AND
	    usage_metric_date BETWEEN $2 AND $3`

	result := &types.UsageMetric{
		RootSpaceID: rootSpaceID,
	}

	startTime := time.UnixMilli(start)
	endTime := time.UnixMilli(end)

	err := s.db.QueryRowContext(
		ctx,
		sqlQuery,
		rootSpaceID,
		s.Date(startTime),
		s.Date(endTime),
	).Scan(
		&result.BandwidthOut,
		&result.BandwidthIn,
		&result.StorageTotal,
		&result.LFSStorageTotal,
		&result.Pushes,
	)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to get metric")
	}

	return result, nil
}

func (s *UsageMetricsStore) List(
	ctx context.Context,
	start int64,
	end int64,
) ([]types.UsageMetric, error) {
	const sqlQuery = `
	SELECT
		usage_metric_space_id,
		COALESCE(SUM(usage_metric_bandwidth_out), 0) AS usage_metric_bandwidth_out,
		COALESCE(SUM(usage_metric_bandwidth_in), 0) AS usage_metric_bandwidth_in,
		COALESCE(AVG(usage_metric_storage_total), 0) AS usage_metric_storage_total,
		COALESCE(AVG(usage_metric_lfs_storage_total), 0) AS usage_metric_lfs_storage_total,
		COALESCE(SUM(usage_metric_pushes), 0) AS usage_metric_pushes
	FROM usage_metrics
	WHERE
	    usage_metric_date BETWEEN $1 AND $2
	GROUP BY usage_metric_space_id
	ORDER BY 
		usage_metric_bandwidth_out DESC,
		usage_metric_bandwidth_in DESC,
		usage_metric_storage_total DESC,
		usage_metric_lfs_storage_total DESC,
		usage_metric_pushes DESC`

	startTime := time.UnixMilli(start)
	endTime := time.UnixMilli(end)

	db := dbtx.GetAccessor(ctx, s.db)
	rows, err := db.QueryContext(ctx, sqlQuery, s.Date(startTime), s.Date(endTime))
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list usage metrics")
	}
	defer rows.Close()

	results := make([]types.UsageMetric, 0, 16)
	for rows.Next() {
		metric := types.UsageMetric{}
		err = rows.Scan(
			&metric.RootSpaceID,
			&metric.BandwidthOut,
			&metric.BandwidthIn,
			&metric.StorageTotal,
			&metric.LFSStorageTotal,
			&metric.Pushes,
		)
		if err != nil {
			return nil, database.ProcessSQLErrorf(ctx, err, "failed to scan usage_metrics")
		}
		results = append(results, metric)
	}

	if err = rows.Err(); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list usage_metrics")
	}
	return results, nil
}

func (s *UsageMetricsStore) Date(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).UnixMilli()
}
