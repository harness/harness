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

package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestUsageMetricsStore_Upsert(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	metricsStore := database.NewUsageMetricsStore(db)
	// First write will set bandwidth and storage to 100
	err := metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:     1,
		BandwidthOut:    100,
		BandwidthIn:     100,
		StorageTotal:    100,
		LFSStorageTotal: 100,
		Pushes:          21,
	})
	require.NoError(t, err)

	// second write will increase bandwidth for 100 and storage remains the same
	err = metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:  1,
		BandwidthOut: 100,
		BandwidthIn:  100,
		Pushes:       3,
	})
	require.NoError(t, err)

	row := db.QueryRowContext(
		ctx,
		`SELECT
    		usage_metric_space_id,
    		usage_metric_date,
    		usage_metric_bandwidth_out,
    		usage_metric_bandwidth_in,
    		usage_metric_storage_total,
    		usage_metric_lfs_storage_total,
			usage_metric_pushes
    	FROM usage_metrics
    	WHERE usage_metric_space_id = ?
    	LIMIT 1`,
		1,
	)
	metric := types.UsageMetric{}
	var date int64
	err = row.Scan(
		&metric.RootSpaceID,
		&date,
		&metric.BandwidthOut,
		&metric.BandwidthIn,
		&metric.StorageTotal,
		&metric.LFSStorageTotal,
		&metric.Pushes,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), metric.RootSpaceID)
	require.Equal(t, metricsStore.Date(time.Now()), date)
	require.Equal(t, int64(200), metric.BandwidthOut)
	require.Equal(t, int64(200), metric.BandwidthIn)
	require.Equal(t, int64(100), metric.StorageTotal)
	require.Equal(t, int64(100), metric.LFSStorageTotal)
	require.Equal(t, int64(24), metric.Pushes)
}

func TestUsageMetricsStore_UpsertOptimistic(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	metricsStore := database.NewUsageMetricsStore(db)

	g, _ := errgroup.WithContext(ctx)

	for range 100 {
		g.Go(func() error {
			return metricsStore.UpsertOptimistic(ctx, &types.UsageMetric{
				RootSpaceID:  1,
				BandwidthOut: 100,
				BandwidthIn:  100,
				Pushes:       21,
			})
		})
	}

	err := g.Wait()
	require.NoError(t, err)

	now := time.Now().UnixMilli()
	metric, err := metricsStore.GetMetrics(ctx, 1, now, now)
	require.NoError(t, err)

	require.Equal(t, int64(100*100), metric.BandwidthOut)
	require.Equal(t, int64(100*100), metric.BandwidthIn)
	require.Equal(t, int64(0), metric.StorageTotal)
	require.Equal(t, int64(0), metric.LFSStorageTotal)
	require.Equal(t, int64(21*100), metric.Pushes)
}

func TestUsageMetricsStore_GetMetrics(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	metricsStore := database.NewUsageMetricsStore(db)
	// First write will set bandwidth and storage to 100
	err := metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:     1,
		BandwidthOut:    100,
		BandwidthIn:     100,
		StorageTotal:    100,
		LFSStorageTotal: 100,
		Pushes:          21,
	})
	require.NoError(t, err)

	now := time.Now().UnixMilli()

	metric, err := metricsStore.GetMetrics(ctx, 1, now, now)
	require.NoError(t, err)

	require.Equal(t, int64(1), metric.RootSpaceID, "expected spaceID = %d, got %d", 1, metric.RootSpaceID)
	require.Equal(t, int64(100), metric.BandwidthOut, "expected bandwidth out = %d, got %d", 100, metric.BandwidthOut)
	require.Equal(t, int64(100), metric.BandwidthIn, "expected bandwidth in = %d, got %d", 100, metric.BandwidthIn)
	require.Equal(t, int64(100), metric.StorageTotal, "expected storage = %d, got %d", 100, metric.StorageTotal)
	require.Equal(t, int64(100), metric.LFSStorageTotal, "expected lfs storage = %d, got %d", 100, metric.LFSStorageTotal)
	require.Equal(t, int64(21), metric.Pushes, "expected pushes = %d, got %d", 21, metric.Pushes)
}

func TestUsageMetricsStore_List(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 2, 0)

	metricsStore := database.NewUsageMetricsStore(db)
	err := metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:     1,
		BandwidthOut:    100,
		BandwidthIn:     100,
		StorageTotal:    100,
		LFSStorageTotal: 100,
		Pushes:          21,
	})
	require.NoError(t, err)

	err = metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:     1,
		BandwidthOut:    50,
		BandwidthIn:     50,
		StorageTotal:    50,
		LFSStorageTotal: 50,
		Pushes:          21,
	})
	require.NoError(t, err)

	err = metricsStore.Upsert(ctx, &types.UsageMetric{
		RootSpaceID:     2,
		BandwidthOut:    200,
		BandwidthIn:     200,
		StorageTotal:    200,
		LFSStorageTotal: 200,
		Pushes:          21,
	})
	require.NoError(t, err)

	now := time.Now().UnixMilli()
	metrics, err := metricsStore.List(ctx, now, now)
	require.NoError(t, err)
	require.Equal(t, 2, len(metrics))

	// list use desc order so first row should be spaceID = 2
	require.Equal(t, int64(2), metrics[0].RootSpaceID)
	require.Equal(t, int64(200), metrics[0].BandwidthOut)
	require.Equal(t, int64(200), metrics[0].BandwidthIn)
	require.Equal(t, int64(200), metrics[0].StorageTotal)
	require.Equal(t, int64(200), metrics[0].LFSStorageTotal)
	require.Equal(t, int64(21), metrics[0].Pushes)

	// second row should be spaceID = 1
	require.Equal(t, int64(1), metrics[1].RootSpaceID)
	require.Equal(t, int64(150), metrics[1].BandwidthOut)
	require.Equal(t, int64(150), metrics[1].BandwidthIn)
	require.Equal(t, int64(50), metrics[1].StorageTotal)
	require.Equal(t, int64(50), metrics[1].LFSStorageTotal)
	require.Equal(t, int64(42), metrics[1].Pushes)
}
