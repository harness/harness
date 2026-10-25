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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIModelStore(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIModelStore(db)

	m := &types.AIModel{
		ID: "vendor/model-a:free", Kind: enum.AIModelKindDecision, ContextLength: 8192,
		SupportsTools: true, Status: enum.AIModelStatusActive,
		SuccessRateByTask: map[string]float64{"reachable": 0.9}, P95LatencyMS: 1200, LastSeen: 42,
	}
	require.NoError(t, s.Upsert(ctx, m))
	require.NotZero(t, m.Created)

	got, err := s.Find(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, got.ID)
	assert.True(t, got.SupportsTools)
	assert.Equal(t, 0.9, got.SuccessRateByTask["reachable"])

	// upsert updates in place and keeps created
	created := got.Created
	m.ContextLength = 16384
	m.Created = 0
	require.NoError(t, s.Upsert(ctx, m))
	got, err = s.Find(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, 16384, got.ContextLength)
	assert.Equal(t, created, got.Created)

	require.NoError(t, s.Upsert(ctx, &types.AIModel{
		ID: "vendor/model-b:free", Kind: enum.AIModelKindChat, Status: enum.AIModelStatusActive,
	}))
	require.NoError(t, s.UpdateStatus(ctx, "vendor/model-b:free", enum.AIModelStatusQuarantined, "paid"))

	all, err := s.List(ctx, types.AIModelFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	q, err := s.List(ctx, types.AIModelFilter{Status: enum.AIModelStatusQuarantined})
	require.NoError(t, err)
	require.Len(t, q, 1)
	assert.Equal(t, "paid", q[0].QuarantinedReason)
	d, err := s.List(ctx, types.AIModelFilter{Kind: enum.AIModelKindDecision})
	require.NoError(t, err)
	assert.Len(t, d, 1)

	_, err = s.Find(ctx, "missing")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	require.ErrorIs(t, s.UpdateStatus(ctx, "missing", enum.AIModelStatusRetired, ""), gitnessstore.ErrResourceNotFound)
}

func TestAIDecisionStore(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIDecisionStore(db)

	taskID := int64(7)
	d := &types.AIDecision{
		TaskID: &taskID, QuestionID: "reachable", ModelBuild: "b1", StateHash: "h1",
		Answer: types.AIDecisionAnswer{
			Value: json.RawMessage(`true`), Probabilities: map[string]float64{"true": 0.8}, Confidence: 0.8,
		},
		Fallback: true,
	}
	require.NoError(t, s.Create(ctx, d))
	require.NotZero(t, d.ID)
	// decision without a task
	require.NoError(t, s.Create(ctx, &types.AIDecision{
		QuestionID: "q", ModelBuild: "b1", StateHash: "h2",
		Answer: types.AIDecisionAnswer{Value: json.RawMessage(`"x"`)},
	}))

	got, err := s.Find(ctx, d.ID)
	require.NoError(t, err)
	require.NotNil(t, got.TaskID)
	assert.Equal(t, taskID, *got.TaskID)
	assert.True(t, got.Fallback)
	assert.Nil(t, got.Outcome)
	assert.Equal(t, 0.8, got.Answer.Confidence)
	assert.JSONEq(t, `true`, string(got.Answer.Value))

	require.NoError(t, s.SetOutcome(ctx, d.ID, enum.AIDecisionOutcomeOverridden))
	list, err := s.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].Outcome)
	assert.Equal(t, enum.AIDecisionOutcomeOverridden, *list[0].Outcome)

	require.ErrorIs(t, s.SetOutcome(ctx, 9999, enum.AIDecisionOutcomeSuccess), gitnessstore.ErrResourceNotFound)
}

func TestAIQuotaStore_IncrementQuota(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIQuotaStore(db)

	q, err := s.Find(ctx, "2026-01-01", "decision")
	require.NoError(t, err)
	assert.Zero(t, q.Used)

	for i := 0; i < 3; i++ {
		ok, err := s.IncrementQuota(ctx, "2026-01-01", "decision", 3)
		require.NoError(t, err)
		assert.True(t, ok, "increment %d", i)
	}
	ok, err := s.IncrementQuota(ctx, "2026-01-01", "decision", 3)
	require.NoError(t, err)
	assert.False(t, ok)

	q, err = s.Find(ctx, "2026-01-01", "decision")
	require.NoError(t, err)
	assert.Equal(t, 3, q.Used)

	// other class and other day are independent
	ok, err = s.IncrementQuota(ctx, "2026-01-01", "chat", 1)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = s.IncrementQuota(ctx, "2026-01-02", "decision", 1)
	require.NoError(t, err)
	assert.True(t, ok)

	// non-positive limit never inserts a row
	ok, err = s.IncrementQuota(ctx, "2026-01-03", "decision", 0)
	require.NoError(t, err)
	assert.False(t, ok)
	q, err = s.Find(ctx, "2026-01-03", "decision")
	require.NoError(t, err)
	assert.Zero(t, q.Used)
}

func TestAIQuotaStore_IncrementQuotaConcurrent(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIQuotaStore(db)

	const limit = 5
	var granted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.IncrementQuota(ctx, "2026-02-01", "decision", limit)
			if err == nil && ok {
				granted.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.EqualValues(t, limit, granted.Load())
	q, err := s.Find(ctx, "2026-02-01", "decision")
	require.NoError(t, err)
	assert.Equal(t, limit, q.Used)
}

func TestAIQuotaStore_DecrementAndRaise(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIQuotaStore(db)

	// missing row: no-op
	require.NoError(t, s.DecrementQuota(ctx, "2026-03-01", "total"))

	ok, err := s.IncrementQuota(ctx, "2026-03-01", "total", 5)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, s.DecrementQuota(ctx, "2026-03-01", "total"))
	require.NoError(t, s.DecrementQuota(ctx, "2026-03-01", "total")) // never below zero
	q, err := s.Find(ctx, "2026-03-01", "total")
	require.NoError(t, err)
	assert.Zero(t, q.Used)

	// raise creates the row and never lowers it
	require.NoError(t, s.RaiseQuota(ctx, "2026-03-02", "total", 5))
	require.NoError(t, s.RaiseQuota(ctx, "2026-03-02", "total", 3))
	q, err = s.Find(ctx, "2026-03-02", "total")
	require.NoError(t, err)
	assert.Equal(t, 5, q.Used)
	ok, err = s.IncrementQuota(ctx, "2026-03-02", "total", 5)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAIPlannerOnSQLiteStore(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	quota := database.NewAIQuotaStore(db)

	// two planners model two replicas sharing the same table; the clock is the real one,
	// so only the invariants that hold at any hour are asserted
	const daily = 10
	a := aigateway.NewStorePlanner(daily, 1000, map[aigateway.Class]float64{aigateway.ClassAgent: 1}, quota)
	b := aigateway.NewStorePlanner(daily, 1000, map[aigateway.Class]float64{aigateway.ClassAgent: 1}, quota)

	var granted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(p *aigateway.Planner) {
			defer wg.Done()
			if p.Reserve(aigateway.ClassAgent) == nil {
				granted.Add(1)
			}
		}([]*aigateway.Planner{a, b}[i%2])
	}
	wg.Wait()
	assert.EqualValues(t, daily, granted.Load())
	assert.Equal(t, daily, a.Snapshot().Used)
	assert.Zero(t, b.Remaining(aigateway.ClassAgent))

	// the usage store accepts what the gateway records
	usage := database.NewAIUsageStore(db)
	require.NoError(t, usage.Create(ctx, &types.AIUsage{Feature: "agent", Model: "m", Requests: 1}))
}

func TestAIUsageStore(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAIUsageStore(db)

	repoID := int64(3)
	require.NoError(t, s.Create(ctx, &types.AIUsage{
		RepoID: &repoID, Feature: "shield", Model: "m", Requests: 1, TokensIn: 10, TokensOut: 5, Created: 100,
	}))
	require.NoError(t, s.Create(ctx, &types.AIUsage{Feature: "chat", Model: "m", Requests: 2, Created: 200}))

	all, err := s.ListSince(ctx, 0)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.NotNil(t, all[0].RepoID)
	assert.Equal(t, repoID, *all[0].RepoID)
	assert.Nil(t, all[1].SpaceID)

	recent, err := s.ListSince(ctx, 150)
	require.NoError(t, err)
	require.Len(t, recent, 1)
	assert.Equal(t, "chat", recent[0].Feature)
	assert.Zero(t, recent[0].CostUSD)
}
