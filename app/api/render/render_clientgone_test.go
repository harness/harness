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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// errClientDisconnected mirrors the unexported sentinel net/http's HTTP/2 server returns on a client
// disconnect. It lives in an internal package, so the message is all that can be matched on.
var errClientDisconnected = errors.New("client disconnected")

type failingWriter struct {
	*httptest.ResponseRecorder
	err error
}

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestIsClientGone(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"broken pipe", fmt.Errorf("write tcp: %w", syscall.EPIPE), true},
		{"connection reset", fmt.Errorf("write tcp: %w", syscall.ECONNRESET), true},
		{"use of closed connection", fmt.Errorf("write tcp: %w", net.ErrClosed), true},
		{"context canceled", fmt.Errorf("copy: %w", context.Canceled), true},
		{"http2 client disconnected", fmt.Errorf("write: %w", errClientDisconnected), true},
		{"deadline exceeded is not a disconnect", context.DeadlineExceeded, false},
		{"encoding failure", errors.New("json: unsupported type: chan int"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClientGone(tt.err); got != tt.want {
				t.Fatalf("isClientGone(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// writeJSON logs through the global logger, so these subtests must not run in parallel.
func TestWriteJSON_ClientGoneLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantLevel string
	}{
		{"broken pipe is debug", fmt.Errorf("write: %w", syscall.EPIPE), "debug"},
		{"other failure stays an error", errors.New("disk full"), "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := log.Logger
			log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)
			defer func() { log.Logger = old }()

			writeJSON(failingWriter{httptest.NewRecorder(), tt.err}, map[string]string{"a": "b"})

			if want := fmt.Sprintf(`"level":"%s"`, tt.wantLevel); !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Fatalf("log = %s, want level %s", buf.String(), tt.wantLevel)
			}
		})
	}
}

func TestReader_ClientGoneLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantLevel string
	}{
		{"client disconnect is debug", fmt.Errorf("read: %w", syscall.ECONNRESET), "debug"},
		{"other failure stays an error", errors.New("storage unavailable"), "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			ctx := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(context.Background())

			Reader(ctx, httptest.NewRecorder(), 200, failingReader{err: tt.err})

			if want := fmt.Sprintf(`"level":"%s"`, tt.wantLevel); !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Fatalf("log = %s, want level %s", buf.String(), tt.wantLevel)
			}
		})
	}
}

var _ io.Reader = failingReader{}
