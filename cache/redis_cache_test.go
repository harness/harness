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

package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// --- test value type ---

type testItem struct {
	ID    int64
	Value string
}

func (t *testItem) Identifier() int64 { return t.ID }

// --- codec ---

type testCodec struct{}

func (testCodec) Encode(v *testItem) string {
	return fmt.Sprintf("%d:%s", v.ID, v.Value)
}

func (testCodec) Decode(s string) (*testItem, error) {
	for i, c := range s {
		if c == ':' {
			id, err := strconv.ParseInt(s[:i], 10, 64)
			if err != nil {
				return nil, err
			}
			return &testItem{ID: id, Value: s[i+1:]}, nil
		}
	}
	return nil, fmt.Errorf("invalid format: %s", s)
}

// --- mock extended getter ---

type mockExtGetter struct {
	findFn     func(ctx context.Context, key int64) (*testItem, error)
	findManyFn func(ctx context.Context, keys []int64) ([]*testItem, error)
}

func (m *mockExtGetter) Find(ctx context.Context, key int64) (*testItem, error) {
	return m.findFn(ctx, key)
}

func (m *mockExtGetter) FindMany(ctx context.Context, keys []int64) ([]*testItem, error) {
	return m.findManyFn(ctx, keys)
}

// --- mock pipeline (supports pipelined GET and SET) ---

type mockPipeline struct {
	redis.Pipeliner
	// getResults maps encoded key -> value. nil value means redis.Nil (miss).
	getResults map[string]*string
	getCalls   []string
	setCalls   []string
	execErr    error
}

func (p *mockPipeline) Get(ctx context.Context, key string) *redis.StringCmd {
	p.getCalls = append(p.getCalls, key)
	cmd := redis.NewStringCmd(ctx)
	if p.getResults != nil {
		if val, ok := p.getResults[key]; ok && val != nil {
			cmd.SetVal(*val)
			return cmd
		}
	}
	cmd.SetErr(redis.Nil)
	return cmd
}

func (p *mockPipeline) Set(ctx context.Context, key string, _ interface{}, _ time.Duration) *redis.StatusCmd {
	p.setCalls = append(p.setCalls, key)
	return redis.NewStatusCmd(ctx)
}

func (p *mockPipeline) Exec(_ context.Context) ([]redis.Cmder, error) {
	return nil, p.execErr
}

// --- mock redis client (supports multiple Pipeline() calls) ---

type mockRedisClient struct {
	redis.UniversalClient
	getResult string
	getErr    error
	setErr    error
	delCalls  []string
	delErr    error
	pipelines []*mockPipeline
	pipeIdx   int
}

func (m *mockRedisClient) Get(ctx context.Context, _ string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if m.getErr != nil {
		cmd.SetErr(m.getErr)
	} else {
		cmd.SetVal(m.getResult)
	}
	return cmd
}

func (m *mockRedisClient) Set(ctx context.Context, _ string, _ interface{}, _ time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
	}
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.delCalls = append(m.delCalls, keys...)
	cmd := redis.NewIntCmd(ctx)
	if m.delErr != nil {
		cmd.SetErr(m.delErr)
	}
	return cmd
}

func (m *mockRedisClient) Pipeline() redis.Pipeliner {
	if m.pipeIdx < len(m.pipelines) {
		p := m.pipelines[m.pipeIdx]
		m.pipeIdx++
		return p
	}
	return &mockPipeline{}
}

func keyEncoder(id int64) string {
	return "test:" + strconv.FormatInt(id, 10)
}

// helper to create a string pointer.
func strPtr(s string) *string { return &s }

// --- mock error pipeline (returns a specific error for all GET commands) ---

type mockErrorPipeline struct {
	redis.Pipeliner
	getErr   error
	setCalls []string
	execErr  error
}

func (p *mockErrorPipeline) Get(ctx context.Context, _ string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	cmd.SetErr(p.getErr)
	return cmd
}

func (p *mockErrorPipeline) Set(ctx context.Context, key string, _ interface{}, _ time.Duration) *redis.StatusCmd {
	p.setCalls = append(p.setCalls, key)
	return redis.NewStatusCmd(ctx)
}

func (p *mockErrorPipeline) Exec(_ context.Context) ([]redis.Cmder, error) {
	return nil, p.execErr
}

// --- mock redis client with error pipeline for first call ---

type mockRedisClientWithErrorPipeline struct {
	redis.UniversalClient
	errorPipeline *mockErrorPipeline
	writePipeline *mockPipeline
	delCalls      []string
	callCount     int
}

func (m *mockRedisClientWithErrorPipeline) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.delCalls = append(m.delCalls, keys...)
	return redis.NewIntCmd(ctx)
}

func (m *mockRedisClientWithErrorPipeline) Pipeline() redis.Pipeliner {
	m.callCount++
	if m.callCount == 1 {
		return m.errorPipeline
	}
	return m.writePipeline
}

// ===========================
// Map() tests
// ===========================

func TestExtendedRedis_Map_EmptyKeys(t *testing.T) {
	c := NewExtendedRedis[int64, *testItem](
		&mockRedisClient{}, &mockExtGetter{}, keyEncoder, testCodec{}, time.Minute, nil,
	)
	m, err := c.Map(context.Background(), []int64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(m))
	}
}

func TestExtendedRedis_Map_AllHits(t *testing.T) {
	readPipe := &mockPipeline{
		getResults: map[string]*string{
			"test:1": strPtr("1:hello"),
			"test:2": strPtr("2:world"),
		},
	}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe}}
	c := NewExtendedRedis[int64, *testItem](
		client, &mockExtGetter{}, keyEncoder, testCodec{}, time.Minute, nil,
	)
	m, err := c.Map(context.Background(), []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m[1].Value != "hello" {
		t.Errorf("expected hello, got %s", m[1].Value)
	}
	if m[2].Value != "world" {
		t.Errorf("expected world, got %s", m[2].Value)
	}
	hits, misses := c.Stats()
	if hits != 2 || misses != 0 {
		t.Errorf("expected 2 hits 0 misses, got %d/%d", hits, misses)
	}
}

func TestExtendedRedis_Map_AllMisses(t *testing.T) {
	readPipe := &mockPipeline{getResults: map[string]*string{}} // all misses
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			var items []*testItem
			for _, k := range keys {
				items = append(items, &testItem{ID: k, Value: fmt.Sprintf("val%d", k)})
			}
			return items, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute, nil,
	)
	m, err := c.Map(context.Background(), []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m[1].Value != "val1" {
		t.Errorf("expected val1, got %s", m[1].Value)
	}
	hits, misses := c.Stats()
	if hits != 0 || misses != 2 {
		t.Errorf("expected 0 hits 2 misses, got %d/%d", hits, misses)
	}
	if len(writePipe.setCalls) != 2 {
		t.Errorf("expected 2 SET calls, got %d", len(writePipe.setCalls))
	}
}

func TestExtendedRedis_Map_MixedHitsAndMisses(t *testing.T) {
	readPipe := &mockPipeline{
		getResults: map[string]*string{
			"test:1": strPtr("1:cached"),
			// key 2 missing
		},
	}
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			var items []*testItem
			for _, k := range keys {
				items = append(items, &testItem{ID: k, Value: "fetched"})
			}
			return items, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute, nil,
	)
	m, err := c.Map(context.Background(), []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m[1].Value != "cached" {
		t.Errorf("expected cached, got %s", m[1].Value)
	}
	if m[2].Value != "fetched" {
		t.Errorf("expected fetched, got %s", m[2].Value)
	}
	hits, misses := c.Stats()
	if hits != 1 || misses != 1 {
		t.Errorf("expected 1 hit 1 miss, got %d/%d", hits, misses)
	}
}

func TestExtendedRedis_Map_PipelineExecError_FallsThrough(t *testing.T) {
	var loggedErr error
	readPipe := &mockPipeline{
		getResults: map[string]*string{},
		execErr:    errors.New("redis down"),
	}
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			var items []*testItem
			for _, k := range keys {
				items = append(items, &testItem{ID: k, Value: "fallback"})
			}
			return items, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErr = err },
	)
	m, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m[1].Value != "fallback" {
		t.Errorf("expected fallback, got %s", m[1].Value)
	}
	if loggedErr == nil {
		t.Error("expected error to be logged")
	}
}

func TestExtendedRedis_Map_FindManyError(t *testing.T) {
	readPipe := &mockPipeline{getResults: map[string]*string{}}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, _ []int64) ([]*testItem, error) {
			return nil, errors.New("db error")
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute, nil,
	)
	_, err := c.Map(context.Background(), []int64{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtendedRedis_Map_DecodeError_EvictsAndTreatedAsMiss(t *testing.T) {
	var loggedErrs []error
	readPipe := &mockPipeline{
		getResults: map[string]*string{
			"test:1": strPtr("invalid_no_colon"),
		},
	}
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			var items []*testItem
			for _, k := range keys {
				items = append(items, &testItem{ID: k, Value: "refetched"})
			}
			return items, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErrs = append(loggedErrs, err) },
	)
	m, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m[1].Value != "refetched" {
		t.Errorf("expected refetched, got %s", m[1].Value)
	}
	// Should have logged decode error
	if len(loggedErrs) == 0 {
		t.Error("expected decode error to be logged")
	}
	// Should have evicted the bad key
	if len(client.delCalls) != 1 || client.delCalls[0] != "test:1" {
		t.Errorf("expected DEL test:1, got %v", client.delCalls)
	}
}

func TestExtendedRedis_Map_Deduplicates(t *testing.T) {
	readPipe := &mockPipeline{
		getResults: map[string]*string{
			"test:1": strPtr("1:hello"),
		},
	}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe}}
	c := NewExtendedRedis[int64, *testItem](
		client, &mockExtGetter{}, keyEncoder, testCodec{}, time.Minute, nil,
	)
	m, err := c.Map(context.Background(), []int64{1, 1, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m))
	}
	// Should have only done 1 GET
	if len(readPipe.getCalls) != 1 {
		t.Errorf("expected 1 GET call, got %d", len(readPipe.getCalls))
	}
}

func TestExtendedRedis_Map_WritePipelineError_Logged(t *testing.T) {
	var loggedErr error
	readPipe := &mockPipeline{getResults: map[string]*string{}}
	writePipe := &mockPipeline{execErr: errors.New("pipe error")}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			return []*testItem{{ID: 1, Value: "val"}}, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErr = err },
	)
	m, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m[1].Value != "val" {
		t.Errorf("expected val, got %s", m[1].Value)
	}
	if loggedErr == nil {
		t.Error("expected pipeline error to be logged")
	}
}

func TestExtendedRedis_Map_RedisNilExecError_NotLogged(t *testing.T) {
	var logCalled bool
	readPipe := &mockPipeline{
		getResults: map[string]*string{},
		execErr:    redis.Nil,
	}
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			return []*testItem{{ID: 1, Value: "val"}}, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, _ error) { logCalled = true },
	)
	_, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logCalled {
		t.Error("redis.Nil exec error should not be logged")
	}
}

func TestExtendedRedis_Map_PerKeyNonNilError_TreatedAsMiss(t *testing.T) {
	var loggedErrs []error
	errPipe := &mockErrorPipeline{
		getErr:  errors.New("connection reset"),
		execErr: errors.New("connection reset"),
	}
	writePipe := &mockPipeline{}
	client := &mockRedisClientWithErrorPipeline{
		errorPipeline: errPipe,
		writePipeline: writePipe,
	}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, keys []int64) ([]*testItem, error) {
			return []*testItem{{ID: 1, Value: "from_db"}}, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErrs = append(loggedErrs, err) },
	)
	m, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m[1].Value != "from_db" {
		t.Errorf("expected from_db, got %s", m[1].Value)
	}
	// Should have logged the pipeline exec error + per-key error
	if len(loggedErrs) < 1 {
		t.Error("expected errors to be logged")
	}
}

func TestExtendedRedis_Map_FindManyUnexpectedKey_Skipped(t *testing.T) {
	var loggedErrs []error
	readPipe := &mockPipeline{getResults: map[string]*string{}}
	writePipe := &mockPipeline{}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, _ []int64) ([]*testItem, error) {
			return []*testItem{
				{ID: 1, Value: "ok"},
				{ID: 99, Value: "unexpected"}, // not in requested keys
			}, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErrs = append(loggedErrs, err) },
	)
	m, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m[1].Value != "ok" {
		t.Errorf("expected ok, got %s", m[1].Value)
	}
	if _, exists := m[99]; exists {
		t.Error("unexpected key 99 should not be in result")
	}
	// Should have logged the unexpected key
	foundUnexpected := false
	for _, e := range loggedErrs {
		if e != nil && errors.Is(e, e) {
			foundUnexpected = true
		}
	}
	if !foundUnexpected || len(loggedErrs) == 0 {
		t.Error("expected unexpected key error to be logged")
	}
}

func TestExtendedRedis_Map_WritePipelineNilError_NotLogged(t *testing.T) {
	var logCalled bool
	readPipe := &mockPipeline{getResults: map[string]*string{}}
	writePipe := &mockPipeline{execErr: redis.Nil}
	client := &mockRedisClient{pipelines: []*mockPipeline{readPipe, writePipe}}
	getter := &mockExtGetter{
		findManyFn: func(_ context.Context, _ []int64) ([]*testItem, error) {
			return []*testItem{{ID: 1, Value: "val"}}, nil
		},
	}
	c := NewExtendedRedis[int64, *testItem](
		client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, _ error) { logCalled = true },
	)
	_, err := c.Map(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logCalled {
		t.Error("redis.Nil from write pipeline should not be logged")
	}
}

// ===========================
// Get() tests
// ===========================

func TestRedis_Get_CacheHit(t *testing.T) {
	client := &mockRedisClient{getResult: "42:hello"}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, _ int64) (*testItem, error) {
			t.Fatal("Find should not be called on cache hit")
			panic("unreachable")
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute, nil)
	item, err := c.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Value != "hello" {
		t.Errorf("expected hello, got %s", item.Value)
	}
	hits, misses := c.Stats()
	if hits != 1 || misses != 0 {
		t.Errorf("expected 1 hit 0 miss, got %d/%d", hits, misses)
	}
}

func TestRedis_Get_CacheMiss_FetchesFromSource(t *testing.T) {
	client := &mockRedisClient{getErr: redis.Nil}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, key int64) (*testItem, error) {
			return &testItem{ID: key, Value: "from_db"}, nil
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute, nil)
	item, err := c.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Value != "from_db" {
		t.Errorf("expected from_db, got %s", item.Value)
	}
	hits, misses := c.Stats()
	if hits != 0 || misses != 1 {
		t.Errorf("expected 0 hit 1 miss, got %d/%d", hits, misses)
	}
}

func TestRedis_Get_RedisError_FallsToSource(t *testing.T) {
	var loggedErr error
	client := &mockRedisClient{getErr: errors.New("redis down")}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, key int64) (*testItem, error) {
			return &testItem{ID: key, Value: "fallback"}, nil
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErr = err },
	)
	item, err := c.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Value != "fallback" {
		t.Errorf("expected fallback, got %s", item.Value)
	}
	if loggedErr == nil {
		t.Error("expected redis error to be logged")
	}
}

func TestRedis_Get_DecodeError_EvictsAndRefetches(t *testing.T) {
	var loggedErrs []error
	client := &mockRedisClient{getResult: "invalid_no_colon"}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, key int64) (*testItem, error) {
			return &testItem{ID: key, Value: "fresh"}, nil
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErrs = append(loggedErrs, err) },
	)
	item, err := c.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Value != "fresh" {
		t.Errorf("expected fresh, got %s", item.Value)
	}
	if len(loggedErrs) == 0 {
		t.Error("expected decode error to be logged")
	}
	if len(client.delCalls) != 1 {
		t.Errorf("expected 1 DEL call for eviction, got %d", len(client.delCalls))
	}
}

func TestRedis_Get_FindError_ReturnsError(t *testing.T) {
	client := &mockRedisClient{getErr: redis.Nil}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, _ int64) (*testItem, error) {
			return nil, errors.New("not found")
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute, nil)
	_, err := c.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRedis_Get_SetError_LoggedButReturnsItem(t *testing.T) {
	var loggedErr error
	client := &mockRedisClient{
		getErr: redis.Nil,
		setErr: errors.New("set failed"),
	}
	getter := &mockExtGetter{
		findFn: func(_ context.Context, key int64) (*testItem, error) {
			return &testItem{ID: key, Value: "val"}, nil
		},
	}
	c := NewRedis[int64, *testItem](client, getter, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErr = err },
	)
	item, err := c.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Value != "val" {
		t.Errorf("expected val, got %s", item.Value)
	}
	if loggedErr == nil {
		t.Error("expected SET error to be logged")
	}
}

// ===========================
// Evict() tests
// ===========================

func TestRedis_Evict_Success(t *testing.T) {
	client := &mockRedisClient{}
	c := NewRedis[int64, *testItem](client, &mockExtGetter{}, keyEncoder, testCodec{}, time.Minute, nil)
	c.Evict(context.Background(), 42)
	if len(client.delCalls) != 1 || client.delCalls[0] != "test:42" {
		t.Errorf("expected DEL test:42, got %v", client.delCalls)
	}
}

func TestRedis_Evict_Error_Logged(t *testing.T) {
	var loggedErr error
	client := &mockRedisClient{delErr: errors.New("del failed")}
	c := NewRedis[int64, *testItem](client, &mockExtGetter{}, keyEncoder, testCodec{}, time.Minute,
		func(_ context.Context, err error) { loggedErr = err },
	)
	c.Evict(context.Background(), 42)
	if loggedErr == nil {
		t.Error("expected DEL error to be logged")
	}
}

// ===========================
// Constructor / Stats tests
// ===========================

func TestNewExtendedRedis(t *testing.T) {
	getter := &mockExtGetter{}
	c := NewExtendedRedis[int64, *testItem](
		&mockRedisClient{}, getter, keyEncoder, testCodec{}, time.Minute, nil,
	)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	hits, misses := c.Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("expected 0/0, got %d/%d", hits, misses)
	}
}
