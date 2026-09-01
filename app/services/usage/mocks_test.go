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

package usage

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MetricsMock is handed to code that expects a store.UsageMetricStore, so it has
// to keep implementing the full interface.
var _ store.UsageMetricStore = (*MetricsMock)(nil)

func TestMetricsMock_GetLatestStorage(t *testing.T) {
	var gotRootSpaceID int64

	m := &MetricsMock{
		GetLatestStorageFn: func(_ context.Context, rootSpaceID int64) (*types.UsageMetric, bool, error) {
			gotRootSpaceID = rootSpaceID
			return &types.UsageMetric{RootSpaceID: rootSpaceID, StorageTotal: 512}, true, nil
		},
	}

	metric, found, err := m.GetLatestStorage(context.Background(), 3)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, metric)

	assert.Equal(t, int64(3), gotRootSpaceID)
	assert.Equal(t, int64(3), metric.RootSpaceID)
	assert.Equal(t, int64(512), metric.StorageTotal)
}
