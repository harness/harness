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

package render

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// nonFlusher is a ResponseWriter that doesn't implement http.Flusher.
type nonFlusher struct {
	header http.Header
	status int
}

func (n *nonFlusher) Header() http.Header         { return n.header }
func (n *nonFlusher) Write(b []byte) (int, error) { return len(b), nil }
func (n *nonFlusher) WriteHeader(status int)      { n.status = status }

func TestStreamSSE_NotFlushable(t *testing.T) {
	w := &nonFlusher{header: http.Header{}}

	StreamSSE(context.Background(), w, nil, nil, nil)

	require.Equal(t, http.StatusInternalServerError, w.status)
}

func TestStreamSSE_EventsThenClose(t *testing.T) {
	rec := httptest.NewRecorder()
	events := make(chan *sse.Event, 2)
	events <- &sse.Event{Type: enum.SSETypeExecutionRunning, Data: []byte(`{"a":"<b>"}`)}
	events <- &sse.Event{Type: enum.SSETypeExecutionCompleted, Data: []byte(`1`)}
	close(events)

	StreamSSE(context.Background(), rec, nil, events, nil)

	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	require.Equal(t, "no", rec.Header().Get("X-Accel-Buffering"))
	require.True(t, rec.Flushed)

	body := rec.Body.String()
	require.True(t, strings.HasPrefix(body, ": ping\n\n"), body)
	require.Contains(t, body, "event: execution_running\ndata: {\"a\":\"<b>\"}\n\n")
	require.Contains(t, body, "event: execution_completed\ndata: 1\n\n")
	require.True(t, strings.HasSuffix(body, "event: error\ndata: eof\n\n"), body)
}

func TestStreamSSE_StopConditions(t *testing.T) {
	tests := []struct {
		name  string
		setup func() (context.Context, <-chan struct{}, <-chan error)
	}{
		{"stop channel", func() (context.Context, <-chan struct{}, <-chan error) {
			stop := make(chan struct{})
			close(stop)
			return context.Background(), stop, nil
		}},
		{"error channel", func() (context.Context, <-chan struct{}, <-chan error) {
			errCh := make(chan error, 1)
			errCh <- context.Canceled
			return context.Background(), nil, errCh
		}},
		{"context cancelled", func() (context.Context, <-chan struct{}, <-chan error) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, nil, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, stop, errCh := tt.setup()
			rec := httptest.NewRecorder()
			done := make(chan struct{})

			go func() {
				defer close(done)
				StreamSSE(ctx, rec, stop, make(chan *sse.Event), errCh)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("StreamSSE did not return")
			}
			require.Contains(t, rec.Body.String(), "event: error\ndata: eof")
		})
	}
}
