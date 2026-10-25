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

package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/services/aigateway/decisions"
)

const catalogJSON = `{"data":[
 {"id":"baseten/mammoth-1:free","context_length":16384,"pricing":{"prompt":"0","completion":"0"}},
 {"id":"baseten/elephant-1:free","context_length":16384,"pricing":{"prompt":"0","completion":"0"}},
 {"id":"vendor/coder:free","context_length":131072,"pricing":{"prompt":"0","completion":"0","request":"0"},
  "supported_parameters":["tools","structured_outputs"]},
 {"id":"vendor/small-free","context_length":8192,"pricing":{"prompt":"0","completion":"0"}},
 {"id":"vendor/paid","context_length":200000,"pricing":{"prompt":"0.000003","completion":"0.000015"},
  "supported_parameters":["tools"]},
 {"id":"vendor/broken","context_length":200000,"pricing":{"prompt":"abc","completion":"0"}},
 {"id":"openrouter/free","context_length":131072,"pricing":{"prompt":"0","completion":"0"}}
]}`

type fakeOpenRouter struct {
	mu        sync.Mutex
	decisions map[string]func(w http.ResponseWriter) // by model
	chatCost  float64
	requests  []map[string]any
	paths     []string
}

func (f *fakeOpenRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.paths = append(f.paths, r.URL.Path)
	if r.Header.Get("Authorization") != "Bearer test-key" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	f.requests = append(f.requests, body)

	switch r.URL.Path {
	case "/api/v1/models":
		_, _ = w.Write([]byte(catalogJSON))
	case "/api/alpha/decisions":
		model, _ := body["model"].(string)
		if h, ok := f.decisions[model]; ok {
			h(w)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "/api/v1/chat/completions":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "gen-1",
			"model":   body["model"],
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "cost": f.chatCost},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeOpenRouter) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, p := range f.paths {
		if p == path {
			n++
		}
	}
	return n
}

func (f *fakeOpenRouter) last() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

func answer(model string, cost float64) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": model + "-build",
			"answers": map[string]any{
				"fix": map[string]any{"type": "noul", "noul": 0.9},
			},
			"usage": map[string]any{"cost": cost},
		})
	}
}

func status(code int) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) { w.WriteHeader(code) }
}

func newTestGateway(t *testing.T, f *fakeOpenRouter, daily int) *Gateway {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)

	g := New(Config{
		Enabled:           true,
		BaseURL:           srv.URL + "/api",
		APIKey:            "test-key",
		DailyRequestLimit: daily,
	}, &Redactor{})
	if _, err := g.Catalog().Sync(context.Background()); err != nil {
		t.Fatalf("catalog sync failed: %v", err)
	}
	return g
}

var fixQuestion = map[string]decisions.Question{
	"fix": decisions.Noul("Should an agent fix this?", "", ""),
}

func TestCatalogKeepsOnlyFreeModels(t *testing.T) {
	g := newTestGateway(t, &fakeOpenRouter{}, 100)

	for _, id := range []string{"baseten/mammoth-1:free", "vendor/coder:free", "vendor/small-free", "openrouter/free"} {
		if err := g.Guard().Check(id); err != nil {
			t.Errorf("%s should be allowed: %v", id, err)
		}
	}
	for _, id := range []string{"vendor/paid", "vendor/broken", "anthropic/claude-sonnet", "unknown/model:free"} {
		if err := g.Guard().Check(id); !errors.Is(err, ErrModelNotFree) {
			t.Errorf("%s should be rejected, got %v", id, err)
		}
	}

	agents := g.Catalog().Candidates(Requirements{Tools: true, MinContext: 32000})
	if len(agents) != 1 || agents[0].ID != "vendor/coder:free" {
		t.Errorf("unexpected agent candidates: %+v", agents)
	}
	if all := g.Catalog().Candidates(Requirements{}); len(all) != 4 {
		t.Errorf("router must not be a candidate and paid models must be excluded, got %d", len(all))
	}
}

func TestGuardBeforeSyncOnlyAcceptsFreeSuffix(t *testing.T) {
	g := NewGuard()
	if err := g.Check("vendor/x:free"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := g.Check("vendor/x"); !errors.Is(err, ErrModelNotFree) {
		t.Errorf("expected ErrModelNotFree, got %v", err)
	}
}

func TestDecideSendsFreeOnlyProviderAndCaches(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: answer(DefaultDecisionModel, 0),
	}}
	g := newTestGateway(t, f, 100)

	state := map[string]any{"error": "panic: nil pointer", "user": "jane@example.com"}
	res, err := g.Decide(context.Background(), "task-1", state, fixQuestion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Fallback || res.Cached || res.Model != DefaultDecisionModel+"-build" {
		t.Errorf("unexpected result: %+v", res)
	}
	if d := res.Decisions["fix"]; d.Value != "true" || d.Confidence != 0.9 {
		t.Errorf("unexpected decision: %+v", d)
	}

	sent := f.last()
	maxPrice := sent["provider"].(map[string]any)["max_price"].(map[string]any)
	if maxPrice["prompt"] != 0.0 || maxPrice["completion"] != 0.0 {
		t.Errorf("max_price must be zero, got %v", maxPrice)
	}
	if sent["session_id"] != "task-1" {
		t.Errorf("session_id not forwarded: %v", sent["session_id"])
	}
	if user := sent["state"].(map[string]any)["user"]; user != "[REDACTED:email]" {
		t.Errorf("state must be redacted, got %v", user)
	}

	res, err = g.Decide(context.Background(), "task-1", state, fixQuestion)
	if err != nil || !res.Cached {
		t.Fatalf("second call must be served from cache: %+v %v", res, err)
	}
	if n := f.count("/api/alpha/decisions"); n != 1 {
		t.Errorf("expected 1 decisions request, got %d", n)
	}
	if used := g.Quota().UsedClass[ClassDecision]; used != 1 {
		t.Errorf("cache hits must not use quota, used %d", used)
	}
}

func TestDecideFallsBackToElephant(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         status(http.StatusBadGateway),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 100)

	res, err := g.Decide(context.Background(), "", "state", fixQuestion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Fallback || res.Model != DefaultDecisionFallbackModel+"-build" {
		t.Errorf("expected fallback result, got %+v", res)
	}
	// both attempts count against the quota, failed requests included
	if used := g.Quota().Used; used != 2 {
		t.Errorf("expected 2 requests charged, got %d", used)
	}
}

func TestDecideDoesNotFallBackOnRateLimit(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         status(http.StatusTooManyRequests),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 100)

	if _, err := g.Decide(context.Background(), "", "state", fixQuestion); !IsRateLimited(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if n := f.count("/api/alpha/decisions"); n != 1 {
		t.Errorf("rate limits are per account, fallback must not be tried, got %d requests", n)
	}
}

func TestDecideQuarantinesPaidResponse(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         answer(DefaultDecisionModel, 0.0001),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 100)

	res, err := g.Decide(context.Background(), "", "state", fixQuestion)
	if err != nil || !res.Fallback {
		t.Fatalf("expected fallback after paid response: %+v %v", res, err)
	}
	if err := g.Guard().Check(DefaultDecisionModel); !errors.Is(err, ErrModelQuarantine) {
		t.Errorf("paid model must be quarantined, got %v", err)
	}
}

func TestDecideStopsWhenQuotaIsExhausted(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: answer(DefaultDecisionModel, 0),
	}}
	// 4 requests a day: the decision class gets 25%, a single request
	g := newTestGateway(t, f, 4)
	g.Planner().now = fixedClock("2026-10-25T08:00:00Z")

	if _, err := g.Decide(context.Background(), "", "a", fixQuestion); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := g.Decide(context.Background(), "", "b", fixQuestion)
	var qe *QuotaError
	if !errors.As(err, &qe) || !qe.Daily {
		t.Fatalf("expected daily quota error, got %v", err)
	}
	if n := f.count("/api/alpha/decisions"); n != 1 {
		t.Errorf("no request must be sent without quota, got %d", n)
	}
}

func TestChatRejectsPaidModelsAndRedacts(t *testing.T) {
	f := &fakeOpenRouter{}
	g := newTestGateway(t, f, 100)

	_, err := g.Chat(context.Background(), ClassTriage, ChatRequest{
		Model:    "vendor/paid",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if !errors.Is(err, ErrModelNotFree) {
		t.Fatalf("expected ErrModelNotFree, got %v", err)
	}
	if n := f.count("/api/v1/chat/completions"); n != 0 {
		t.Fatalf("no request must reach OpenRouter for a paid model")
	}

	resp, err := g.Chat(context.Background(), ClassTriage, ChatRequest{
		Model:    "vendor/coder:free",
		Messages: []ChatMessage{{Role: "user", Content: "explain: password=hunter22 from 10.0.0.12"}},
	})
	if err != nil || resp.Content() != "ok" {
		t.Fatalf("unexpected result: %+v %v", resp, err)
	}

	sent := f.last()
	content := sent["messages"].([]any)[0].(map[string]any)["content"].(string)
	if strings.Contains(content, "hunter22") || strings.Contains(content, "10.0.0.12") {
		t.Errorf("message was not redacted: %s", content)
	}
	if sent["usage"].(map[string]any)["include"] != true {
		t.Errorf("usage accounting must be requested to verify cost")
	}
}

func TestChatQuarantinesOnCost(t *testing.T) {
	f := &fakeOpenRouter{chatCost: 0.01}
	g := newTestGateway(t, f, 100)

	req := ChatRequest{Model: "vendor/coder:free", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	if _, err := g.Chat(context.Background(), ClassTriage, req); !errors.Is(err, ErrPaidUsage) {
		t.Fatalf("expected ErrPaidUsage, got %v", err)
	}
	if _, err := g.Chat(context.Background(), ClassTriage, req); !errors.Is(err, ErrModelQuarantine) {
		t.Fatalf("expected ErrModelQuarantine, got %v", err)
	}
}

func TestAgentEnv(t *testing.T) {
	g := newTestGateway(t, &fakeOpenRouter{}, 100)

	env, err := g.AgentEnv("vendor/coder:free")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["ANTHROPIC_MODEL"] != "vendor/coder:free" || env["ANTHROPIC_AUTH_TOKEN"] != "test-key" ||
		!strings.HasSuffix(env["ANTHROPIC_BASE_URL"], "/api") {
		t.Errorf("unexpected env: %v", env)
	}
	if _, err := g.AgentEnv("vendor/small-free"); err == nil {
		t.Errorf("agents need tool calling")
	}
	if _, err := g.AgentEnv("vendor/paid"); !errors.Is(err, ErrModelNotFree) {
		t.Errorf("expected ErrModelNotFree, got %v", err)
	}
}

func TestDisabledGateway(t *testing.T) {
	g := New(Config{}, &Redactor{})
	if _, err := g.Decide(context.Background(), "", "s", fixQuestion); !errors.Is(err, ErrDisabled) {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
	if _, err := g.Chat(context.Background(), ClassManual, ChatRequest{}); !errors.Is(err, ErrDisabled) {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}
