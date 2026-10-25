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

package router

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/ingest"
	"github.com/harness/gitness/app/services/insights/insightstest"
	"github.com/harness/gitness/types"
)

const testdata = "../services/insights/ingest/testdata/"

func newIngestRouter(t *testing.T, rate int) (*InsightsIngestRouter, *insightstest.Store) {
	t.Helper()
	st := insightstest.New()
	st.Projects[42] = &types.InsightsProject{
		ID: 42, RepoID: 1, Identifier: "api", PublicKey: "abc123publickey", Enabled: true,
		RateLimitPerMinute: rate, SampleRate: 1,
	}
	svc := insights.NewService(insights.Config{Limits: ingest.Limits{MaxCompressedSize: 64 << 10}},
		st.ProjectStore(), st.IssueStore(), st.EventStore())
	return NewInsightsIngestRouter(svc), st
}

func do(t *testing.T, r *InsightsIngestRouter, method, target string, body []byte, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	if !r.IsEligibleTraffic(req) {
		t.Fatalf("%s not eligible for the insights router", target)
	}
	rec := httptest.NewRecorder()
	r.Handle(rec, req)
	return rec
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(testdata + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInsightsIngest_EnvelopeFixtures(t *testing.T) {
	want := map[string]string{
		"sentry_go_error.envelope":         "b5a1f7c2e8d34f6a9c0d1e2f3a4b5c6d",
		"sentry_javascript_error.envelope": "0f1e2d3c4b5a69788796a5b4c3d2e1f0",
		"sentry_python_error.envelope":     "c0ffee00c0ffee00c0ffee00c0ffee00",
	}
	for name, id := range want {
		t.Run(name, func(t *testing.T) {
			r, st := newIngestRouter(t, 100)
			rec := do(t, r, http.MethodPost, "/api/42/envelope/", readFixture(t, name), nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
			}
			var out map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out["id"] != id {
				t.Errorf("body = %s, want id %s", rec.Body, id)
			}
			if len(st.Events) != 1 || len(st.Issues) != 1 {
				t.Errorf("events = %d issues = %d", len(st.Events), len(st.Issues))
			}
		})
	}
}

func TestInsightsIngest_AuthSources(t *testing.T) {
	body := readFixture(t, "sentry_go_error.envelope")
	tests := []struct {
		name   string
		target string
		hdr    map[string]string
		want   int
	}{
		{"x-sentry-auth", "/api/42/envelope/", map[string]string{"X-Sentry-Auth": "Sentry sentry_version=7, sentry_key=abc123publickey"}, 200},
		{"authorization header", "/api/42/envelope/", map[string]string{"Authorization": "Sentry sentry_key=abc123publickey"}, 200},
		{"query", "/api/42/envelope/?sentry_key=abc123publickey", nil, 200},
		{"envelope dsn", "/api/42/envelope/", nil, 200},
		{"no trailing slash", "/api/42/envelope", nil, 200},
		{"wrong header key", "/api/42/envelope/", map[string]string{"X-Sentry-Auth": "Sentry sentry_key=wrong"}, 403},
		{"wrong query key", "/api/42/envelope/?sentry_key=wrong", nil, 403},
		{"unknown project", "/api/43/envelope/?sentry_key=abc123publickey", nil, 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newIngestRouter(t, 100)
			rec := do(t, r, http.MethodPost, tt.target, body, tt.hdr)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body)
			}
			if tt.want == http.StatusForbidden && !strings.Contains(rec.Body.String(), "detail") {
				t.Errorf("403 body = %s", rec.Body)
			}
		})
	}
}

func TestInsightsIngest_NoKeyIsForbidden(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	body := []byte("{\"event_id\":\"a\"}\n{\"type\":\"event\"}\n{\"message\":\"x\"}\n")
	if rec := do(t, r, http.MethodPost, "/api/42/envelope/", body, nil); rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestInsightsIngest_LegacyStore(t *testing.T) {
	r, st := newIngestRouter(t, 100)
	rec := do(t, r, http.MethodPost, "/api/42/store/?sentry_key=abc123publickey",
		readFixture(t, "store_legacy.json"), nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "fc6d8c0c43fc4630ad850ee518f1b9d0") {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if len(st.Events) != 1 {
		t.Errorf("events = %d", len(st.Events))
	}
}

func TestInsightsIngest_Gzip(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(readFixture(t, "sentry_python_error.envelope"))
	_ = zw.Close()

	r, st := newIngestRouter(t, 100)
	rec := do(t, r, http.MethodPost, "/api/42/envelope/", buf.Bytes(), map[string]string{"Content-Encoding": "gzip"})
	if rec.Code != http.StatusOK || len(st.Events) != 1 {
		t.Errorf("status = %d events = %d body = %s", rec.Code, len(st.Events), rec.Body)
	}
}

func TestInsightsIngest_RateLimited(t *testing.T) {
	r, _ := newIngestRouter(t, 1)
	body := readFixture(t, "sentry_go_error.envelope")
	if rec := do(t, r, http.MethodPost, "/api/42/envelope/", body, nil); rec.Code != http.StatusOK {
		t.Fatalf("first status = %d", rec.Code)
	}
	// a second event id so it is not a duplicate.
	body = bytes.ReplaceAll(body, []byte("b5a1f7c2e8d34f6a9c0d1e2f3a4b5c6d"), []byte("11111111111111111111111111111111"))
	rec := do(t, r, http.MethodPost, "/api/42/envelope/", body, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	secs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || secs < 1 {
		t.Errorf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	if rec.Header().Get("X-Sentry-Rate-Limits") == "" {
		t.Error("missing X-Sentry-Rate-Limits")
	}
}

func TestInsightsIngest_TooLarge(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	body := bytes.Repeat([]byte("x"), 70<<10)
	if rec := do(t, r, http.MethodPost, "/api/42/envelope/?sentry_key=abc123publickey", body, nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestInsightsIngest_Malformed(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	rec := do(t, r, http.MethodPost, "/api/42/envelope/?sentry_key=abc123publickey", []byte("not an envelope"), nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
}

func TestInsightsIngest_CORSPreflight(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	rec := do(t, r, http.MethodOptions, "/api/42/envelope/", nil, map[string]string{
		"Origin":                         "https://app.example.com",
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "x-sentry-auth,content-type",
	})
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if got := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers")); !strings.Contains(got, "x-sentry-auth") {
		t.Errorf("Allow-Headers = %q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("credentials must not be allowed")
	}
}

func TestInsightsIngest_CORSOnResponse(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	rec := do(t, r, http.MethodPost, "/api/42/envelope/", readFixture(t, "sentry_go_error.envelope"),
		map[string]string{"Origin": "https://app.example.com"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Allow-Origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestInsightsIngest_MethodNotAllowed(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	if rec := do(t, r, http.MethodGet, "/api/42/envelope/", nil, nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestInsightsIngest_Eligibility(t *testing.T) {
	r, _ := newIngestRouter(t, 100)
	tests := map[string]bool{
		"/api/42/envelope/":                 true,
		"/api/42/envelope":                  true,
		"/api/7/store/":                     true,
		"/api/v1/user":                      false,
		"/api/v1/envelope/":                 false,
		"/api/42/envelope/extra":            false,
		"/api/42/other/":                    false,
		"/api/v1/repos/x/insights/projects": false,
		"/v1/envelope/":                     false,
	}
	for path, want := range tests {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if got := r.IsEligibleTraffic(req); got != want {
			t.Errorf("IsEligibleTraffic(%s) = %v, want %v", path, got, want)
		}
	}
}
