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

package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/services/aigateway"
)

const testCatalog = `{"data":[
 {"id":"vendor/coder:free","name":"Coder","context_length":131072,
  "pricing":{"prompt":"0","completion":"0"},"supported_parameters":["tools"]},
 {"id":"vendor/chat:free","name":"Chat","context_length":8192,
  "pricing":{"prompt":"0","completion":"0"}},
 {"id":"vendor/paid","name":"Paid","context_length":200000,
  "pricing":{"prompt":"0.000003","completion":"0.000015"}}
]}`

func newTestGateway(t *testing.T, enabled bool) *aigateway.Gateway {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testCatalog))
	}))
	t.Cleanup(srv.Close)

	return aigateway.New(aigateway.Config{
		Enabled:           enabled,
		BaseURL:           srv.URL,
		APIKey:            "test-key",
		DailyRequestLimit: 100,
		RequestsPerMinute: 10,
	}, &aigateway.Redactor{})
}

func TestAIDisabled(t *testing.T) {
	c := NewController(nil, nil, newTestGateway(t, false))

	models := c.AIModels(context.Background())
	if models.Enabled || len(models.Models) != 0 || models.SyncedAt != nil {
		t.Fatalf("unexpected models output: %+v", models)
	}

	quota := c.AIQuota(context.Background())
	if quota.Enabled || quota.Limit != 0 || len(quota.UsedByClass) != 0 {
		t.Fatalf("unexpected quota output: %+v", quota)
	}

	// a nil gateway behaves like a disabled one
	c = NewController(nil, nil, nil)
	if c.AIModels(context.Background()).Enabled || c.AIQuota(context.Background()).Enabled {
		t.Fatal("nil gateway must report disabled")
	}
}

func TestAIModels(t *testing.T) {
	gw := newTestGateway(t, true)
	if _, err := gw.Catalog().Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	gw.Guard().Quarantine("vendor/chat:free", "test")

	out := NewController(nil, nil, gw).AIModels(context.Background())

	if !out.Enabled || out.SyncedAt == nil {
		t.Fatalf("expected enabled output with synced_at: %+v", out)
	}
	want := []AIModel{
		{ID: "vendor/chat:free", Name: "Chat", ContextLength: 8192, SupportsTools: false, Quarantined: true},
		{ID: "vendor/coder:free", Name: "Coder", ContextLength: 131072, SupportsTools: true, Quarantined: false},
	}
	if len(out.Models) != len(want) {
		t.Fatalf("expected %d models, got %+v", len(want), out.Models)
	}
	for i := range want {
		if out.Models[i] != want[i] {
			t.Errorf("model %d: want %+v got %+v", i, want[i], out.Models[i])
		}
	}
}

func TestAIQuota(t *testing.T) {
	gw := newTestGateway(t, true)
	if err := gw.Planner().Reserve(aigateway.ClassAgent); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	out := NewController(nil, nil, gw).AIQuota(context.Background())

	if !out.Enabled || out.Limit != 100 || out.Used != 1 || out.Day == "" {
		t.Fatalf("unexpected quota output: %+v", out)
	}
	if out.UsedByClass["agent"] != 1 || out.UsedByClass["decision"] != 0 {
		t.Errorf("unexpected used_by_class: %+v", out.UsedByClass)
	}
	for _, class := range []string{"decision", "triage", "agent", "manual"} {
		if _, ok := out.RemainingClass[class]; !ok {
			t.Errorf("missing remaining for class %s", class)
		}
	}
}
