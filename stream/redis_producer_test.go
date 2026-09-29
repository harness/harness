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

package stream

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestIsRetryableRedisError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "eof", err: io.EOF, want: true},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "deadline", err: context.DeadlineExceeded, want: false},
		{name: "loading", err: errors.New("LOADING Redis is loading the dataset in memory"), want: true},
		{name: "readonly", err: errors.New("READONLY You can't write against a read only replica"), want: true},
		{
			name: "wrongtype",
			err:  errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"),
			want: false,
		},
		{name: "net timeout", err: &net.DNSError{IsTimeout: true, Err: "i/o timeout"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableRedisError(tt.err); got != tt.want {
				t.Fatalf("isRetryableRedisError(%v)=%v want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestProducerRetryConfigNormalize(t *testing.T) {
	got := ProducerRetryConfig{}.Normalize()
	if got.MaxAttempts != DefaultProducerMaxAttempts {
		t.Fatalf("MaxAttempts=%d want %d", got.MaxAttempts, DefaultProducerMaxAttempts)
	}
	if got.InitialBackoff != DefaultProducerInitialBackoff {
		t.Fatalf("InitialBackoff=%v want %v", got.InitialBackoff, DefaultProducerInitialBackoff)
	}
	if got.BackoffFactor != DefaultProducerBackoffFactor {
		t.Fatalf("BackoffFactor=%v want %v", got.BackoffFactor, DefaultProducerBackoffFactor)
	}
	if got.MaxBackoff != DefaultProducerMaxBackoff {
		t.Fatalf("MaxBackoff=%v want %v", got.MaxBackoff, DefaultProducerMaxBackoff)
	}

	custom := ProducerRetryConfig{
		MaxAttempts:    2,
		InitialBackoff: 10 * time.Millisecond,
		BackoffFactor:  2,
		MaxBackoff:     20 * time.Millisecond,
	}.Normalize()
	if custom.MaxAttempts != 2 || custom.InitialBackoff != 10*time.Millisecond {
		t.Fatalf("Normalize overwrote custom values: %+v", custom)
	}
}

func TestRedisProducerSend_ReturnsImmediately(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
		MaxRetries:   -1,
	})
	p := NewRedisProducer(rdb, "test", 100, true,
		WithProducerRetry(ProducerRetryConfig{
			MaxAttempts:    2,
			InitialBackoff: time.Millisecond,
			BackoffFactor:  1.5,
			MaxBackoff:     2 * time.Millisecond,
		}),
	)

	start := time.Now()
	msgID, err := p.Send(context.Background(), "events:foo", map[string]any{"k": "v"})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if msgID != "" {
		t.Fatalf("async Send should return empty message id, got %q", msgID)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("Send blocked for %v; expected near-instant goroutine start", elapsed)
	}
	// Let the background goroutine finish so the test process does not race shutdown.
	time.Sleep(100 * time.Millisecond)
}

func TestRedisProducerSendWithRetry_Exhausts(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
		MaxRetries:   -1,
	})
	p := NewRedisProducer(rdb, "test", 100, true,
		WithProducerRetry(ProducerRetryConfig{
			MaxAttempts:    2,
			InitialBackoff: time.Millisecond,
			BackoffFactor:  1.5,
			MaxBackoff:     2 * time.Millisecond,
		}),
	)

	err := p.sendWithRetry(context.Background(), "events:foo", map[string]any{"k": "v"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to write to stream") {
		t.Fatalf("unexpected error: %v", err)
	}
}
