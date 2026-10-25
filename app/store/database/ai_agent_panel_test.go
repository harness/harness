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

package database_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIUsageStore_SumRequestsSince(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	_, err := db.Exec("PRAGMA foreign_keys = OFF")
	require.NoError(t, err)
	ctx := context.Background()
	s := database.NewAIUsageStore(db)

	space, other, repo := int64(1), int64(2), int64(5)
	for _, u := range []*types.AIUsage{
		{SpaceID: &space, Feature: "agent", Model: "m", Requests: 3, Created: 100},
		{SpaceID: &space, Feature: "agent", Model: "m", Requests: 4, Created: 200},
		{RepoID: &repo, Feature: "shield", Model: "m", Requests: 5, Created: 200},
		{SpaceID: &other, Feature: "agent", Model: "m", Requests: 100, Created: 200},
		{Feature: "chat", Model: "m", Requests: 1000, Created: 200},
	} {
		require.NoError(t, s.Create(ctx, u))
	}

	got, err := s.SumRequestsSince(ctx, space, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, 7, got)

	got, err = s.SumRequestsSince(ctx, space, []int64{repo}, 150)
	require.NoError(t, err)
	assert.Equal(t, 9, got)

	got, err = s.SumRequestsSince(ctx, space, []int64{repo}, 1000)
	require.NoError(t, err)
	assert.Zero(t, got)
}

func TestTaskActivityStore_ListLatestByKindForRepos(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	t1, t2, t3 := newTask(1, "one"), newTask(2, "two"), newTask(2, "three")
	for _, tk := range []*types.Task{t1, t2, t3} {
		require.NoError(t, env.tasks.Create(ctx, tk))
	}

	store := database.NewTaskActivityStore(env.db)
	add := func(tk *types.Task, kind enum.TaskActivityKind, created int64) {
		require.NoError(t, store.Create(ctx, &types.TaskActivity{
			TaskID: tk.ID, Kind: kind, Payload: json.RawMessage(`{}`), Created: created,
		}))
	}
	add(t1, enum.TaskActivityKindAgentRun, 10)
	add(t1, enum.TaskActivityKindComment, 15)
	add(t2, enum.TaskActivityKindAgentRun, 30)
	add(t3, enum.TaskActivityKindAgentRun, 20)

	list, err := store.ListLatestByKindForRepos(ctx, []int64{1, 2}, enum.TaskActivityKindAgentRun, 10)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, []int64{30, 20, 10}, []int64{list[0].Created, list[1].Created, list[2].Created})

	list, err = store.ListLatestByKindForRepos(ctx, []int64{2}, enum.TaskActivityKindAgentRun, 1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, t2.ID, list[0].TaskID)

	list, err = store.ListLatestByKindForRepos(ctx, nil, enum.TaskActivityKindAgentRun, 5)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestAITaskStore_ListByTaskIDs(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	db.SetMaxOpenConns(1)
	_, err := db.Exec("PRAGMA foreign_keys = OFF")
	require.NoError(t, err)
	ctx := context.Background()
	s := database.NewAITaskStore(db)

	a, b := int64(7), int64(8)
	for i, tid := range []*int64{&a, &a, &b, nil} {
		tk := newTestAITask([]string{"a1", "a2", "b1", "none"}[i], 3)
		tk.TaskID = tid
		tk.Created = int64(i + 1)
		require.NoError(t, s.Create(ctx, tk))
	}

	list, err := s.ListByTaskIDs(ctx, []int64{a})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "a1", list[0].Identifier)
	assert.Equal(t, "a2", list[1].Identifier)

	list, err = s.ListByTaskIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
}
