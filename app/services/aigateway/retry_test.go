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
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

const chatOKJSON = `{"id":"gen-1","model":"vendor/coder:free",
 "choices":[{"message":{"role":"assistant","content":"ok"}}],
 "usage":{"prompt_tokens":1,"completion_tokens":1,"cost":0}}`

// newRetryGateway serves the free catalog and answers chat requests with respond,
// which gets the 1-based number of the chat request.
func newRetryGateway(t *testing.T, daily int, respond func(n int, w http.ResponseWriter)) (*Gateway, func() int) {
	t.Helper()

	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/models" {
			_, _ = w.Write([]byte(catalogJSON))
			return
		}
		mu.Lock()
		defer mu.Unlock()
		calls++
		respond(calls, w)
	}))
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
	g.Planner().now = fixedClock("2026-10-25T08:00:00Z")

	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
	return g, count
}

// recordSleeps replaces the gateway's sleep and returns the durations it was asked for.
func recordSleeps(g *Gateway) *[]time.Duration {
	var sleeps []time.Duration
	g.sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return &sleeps
}

func chatRequest() ChatRequest {
	return ChatRequest{Model: "vendor/coder:free", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
}

func writeStatus(code int, retryAfter string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(code)
	}
}

func TestRetryTooManyRequestsThenSucceeds(t *testing.T) {
	g, calls := newRetryGateway(t, 100, func(n int, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(chatOKJSON))
	})
	sleeps := recordSleeps(g)

	if _, err := g.Chat(context.Background(), ClassManual, chatRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := calls(); n != 2 {
		t.Errorf("expected 2 requests, got %d", n)
	}
	if used := g.Quota().Used; used != 2 {
		t.Errorf("expected 2 requests charged, got %d", used)
	}
	if len(*sleeps) != 1 || (*sleeps)[0] != time.Second {
		t.Errorf("expected one 1s backoff, got %v", *sleeps)
	}
}

func TestRetryTooManyRequestsExhausted(t *testing.T) {
	g, calls := newRetryGateway(t, 100, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	sleeps := recordSleeps(g)

	_, err := g.Chat(context.Background(), ClassManual, chatRequest())
	if !IsRateLimited(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if n := calls(); n != 3 {
		t.Errorf("expected 3 attempts, got %d", n)
	}
	if used := g.Quota().Used; used != 3 {
		t.Errorf("expected 3 requests charged, got %d", used)
	}
	if want := []time.Duration{time.Second, 2 * time.Second}; len(*sleeps) != 2 ||
		(*sleeps)[0] != want[0] || (*sleeps)[1] != want[1] {
		t.Errorf("expected backoff %v, got %v", want, *sleeps)
	}
}

func TestRetryServerErrorIsRetried(t *testing.T) {
	g, calls := newRetryGateway(t, 100, func(n int, w http.ResponseWriter) {
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(chatOKJSON))
	})
	recordSleeps(g)

	if _, err := g.Chat(context.Background(), ClassManual, chatRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := calls(); n != 3 {
		t.Errorf("expected 3 attempts, got %d", n)
	}
}

func TestRetryBadRequestIsNotRetried(t *testing.T) {
	g, calls := newRetryGateway(t, 100, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
	})
	sleeps := recordSleeps(g)

	_, err := g.Chat(context.Background(), ClassManual, chatRequest())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 error, got %v", err)
	}
	if n := calls(); n != 1 {
		t.Errorf("400 must not be retried, got %d requests", n)
	}
	if len(*sleeps) != 0 {
		t.Errorf("no sleep expected, got %v", *sleeps)
	}
}

func TestRetryAfterIsHonoured(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{name: "seconds", retryAfter: "7", want: 7 * time.Second},
		{name: "capped", retryAfter: "120", want: 30 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := newRetryGateway(t, 100, func(n int, w http.ResponseWriter) {
				if n == 1 {
					writeStatus(http.StatusTooManyRequests, tc.retryAfter)(w)
					return
				}
				_, _ = w.Write([]byte(chatOKJSON))
			})
			sleeps := recordSleeps(g)

			if _, err := g.Chat(context.Background(), ClassManual, chatRequest()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(*sleeps) != 1 || (*sleeps)[0] != tc.want {
				t.Errorf("expected sleep %v, got %v", tc.want, *sleeps)
			}
		})
	}
}

func TestRetryStopsWhenQuotaIsExhausted(t *testing.T) {
	// 10 requests a day: the manual class gets 10%, a single request
	g, calls := newRetryGateway(t, 10, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	recordSleeps(g)

	_, err := g.Chat(context.Background(), ClassManual, chatRequest())
	var qe *QuotaError
	if !errors.As(err, &qe) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if n := calls(); n != 1 {
		t.Errorf("no retry must be sent without quota, got %d requests", n)
	}
}

func TestRetryStopsWhenContextIsCancelled(t *testing.T) {
	g, calls := newRetryGateway(t, 100, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return sleepContext(ctx, time.Hour)
	}

	if _, err := g.Chat(ctx, ClassManual, chatRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if n := calls(); n != 1 {
		t.Errorf("cancelled context must stop retries, got %d requests", n)
	}
}
