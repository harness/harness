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

package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
)

func TestLogPrincipalLoadError(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name      string
		ctx       context.Context
		err       error
		wantLevel string
	}{
		{
			name:      "wrapped context canceled is debug",
			ctx:       context.Background(),
			err:       fmt.Errorf("cache: failed to find one: %w", context.Canceled),
			wantLevel: "debug",
		},
		{
			// The driver reports a cancelled query without wrapping context.Canceled, so the only
			// signal is the request context itself.
			name:      "driver cancel on a cancelled request is debug",
			ctx:       cancelled,
			err:       errors.New("pq: canceling statement due to user request"),
			wantLevel: "debug",
		},
		{
			name:      "deadline exceeded stays an error",
			ctx:       context.Background(),
			err:       fmt.Errorf("cache: %w", context.DeadlineExceeded),
			wantLevel: "error",
		},
		{
			name:      "genuine failure stays an error",
			ctx:       context.Background(),
			err:       errors.New("connection refused"),
			wantLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(tt.ctx)

			logPrincipalLoadError(ctx, tt.err, "failed to load PR author")

			if want := fmt.Sprintf(`"level":"%s"`, tt.wantLevel); !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Fatalf("log = %s, want level %s", buf.String(), tt.wantLevel)
			}
			if !bytes.Contains(buf.Bytes(), []byte("failed to load PR author")) {
				t.Fatalf("message missing from log: %s", buf.String())
			}
		})
	}
}
