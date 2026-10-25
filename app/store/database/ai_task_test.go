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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAITask(identifier string, budget int) *types.AITask {
	return &types.AITask{
		Identifier:     identifier,
		GitspaceConfig: &types.GitspaceConfig{ID: 1, GitspaceInstance: &types.GitspaceInstance{ID: 1}},
		InitialPrompt:  "fix",
		DisplayName:    identifier,
		UserUID:        "u",
		SpaceID:        1,
		Created:        1,
		Updated:        1,
		AIAgent:        enum.AIAgentClaudeCode,
		State:          enum.AITaskStateRunning,
		RequestBudget:  budget,
	}
}

func TestAITaskBudget(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	// the test does not create gitspace/space parents
	db.SetMaxOpenConns(1)
	_, err := db.Exec("PRAGMA foreign_keys = OFF")
	require.NoError(t, err)
	ctx := context.Background()
	s := database.NewAITaskStore(db)

	// defaults
	def := newTestAITask("default", 0)
	require.NoError(t, s.Create(ctx, def))
	got, err := s.Find(ctx, def.ID)
	require.NoError(t, err)
	assert.Equal(t, types.DefaultAITaskRequestBudget, got.RequestBudget)
	assert.Zero(t, got.RequestsUsed)
	assert.Nil(t, got.TaskID)
	assert.Nil(t, got.ModelDecisionID)

	// new fields round trip
	taskID, decisionID := int64(7), int64(9)
	tk := newTestAITask("fields", 3)
	tk.TaskID, tk.ModelDecisionID = &taskID, &decisionID
	require.NoError(t, s.Create(ctx, tk))
	got, err = s.Find(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, got.RequestBudget)
	require.NotNil(t, got.TaskID)
	assert.Equal(t, int64(7), *got.TaskID)
	require.NotNil(t, got.ModelDecisionID)
	assert.Equal(t, int64(9), *got.ModelDecisionID)

	got.RequestBudget = 2
	got.TaskID = nil
	require.NoError(t, s.Update(ctx, got))
	got, err = s.Find(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.RequestBudget)
	assert.Nil(t, got.TaskID)

	// sequential charge up to the budget
	rem, err := s.ChargeRequest(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, rem)
	rem, err = s.ChargeRequest(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, rem)
	_, err = s.ChargeRequest(ctx, tk.ID)
	require.ErrorIs(t, err, store.ErrAITaskBudgetExhausted)
	got, err = s.Find(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.RequestsUsed)

	// update must not reset the counter
	require.NoError(t, s.Update(ctx, got))
	got, err = s.Find(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.RequestsUsed)

	_, err = s.ChargeRequest(ctx, 99999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	// concurrent charges never exceed the budget
	cc := newTestAITask("concurrent", 5)
	require.NoError(t, s.Create(ctx, cc))
	var ok, exhausted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.ChargeRequest(ctx, cc.ID); err == nil {
				ok.Add(1)
			} else if assert.ErrorIs(t, err, store.ErrAITaskBudgetExhausted) {
				exhausted.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(5), ok.Load())
	assert.Equal(t, int32(15), exhausted.Load())
}
