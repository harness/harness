// Copyright 2026 Harness, Inc.
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

package plugin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestCxrPluginList(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	pluginStore := database.NewPluginStore(b.db)
	for i := range 3 {
		require.NoError(t, pluginStore.Create(ctx, &types.Plugin{
			Identifier: fmt.Sprintf("plugin-%d", i), Description: "d", Type: "step", Version: "1", Spec: "spec",
		}))
	}
	c := NewController(pluginStore)

	// A page that is not full needs no count query.
	got, count, err := c.List(ctx, types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}})
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.EqualValues(t, 3, count)

	// A full page triggers the count query.
	got, count, err = c.List(ctx, types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 2}})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.EqualValues(t, 3, count)

	got, count, err = c.List(ctx, types.ListQueryFilter{
		Pagination: types.Pagination{Page: 1, Size: 10}, Query: "plugin-1",
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.EqualValues(t, 1, count)
}

func TestCxrPluginListStoreError(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	pluginStore := database.NewPluginStore(b.db)
	require.NoError(t, b.db.Close())

	_, _, err := NewController(pluginStore).List(ctx, types.ListQueryFilter{
		Pagination: types.Pagination{Page: 1, Size: 10},
	})
	require.ErrorContains(t, err, "failed to list plugins")
}

var errCxrStore = errors.New("store failure")

// cxrPluginStore returns a full page from List and fails Count.
type cxrPluginStore struct {
	store.PluginStore
}

func (cxrPluginStore) List(context.Context, types.ListQueryFilter) ([]*types.Plugin, error) {
	return []*types.Plugin{{Identifier: "a"}, {Identifier: "b"}}, nil
}

func (cxrPluginStore) Count(context.Context, types.ListQueryFilter) (int64, error) {
	return 0, errCxrStore
}

func TestCxrPluginCountError(t *testing.T) {
	_, _, err := NewController(cxrPluginStore{}).List(t.Context(), types.ListQueryFilter{
		Pagination: types.Pagination{Page: 1, Size: 2},
	})
	require.ErrorIs(t, err, errCxrStore)
	require.ErrorContains(t, err, "failed to count plugins")
}
