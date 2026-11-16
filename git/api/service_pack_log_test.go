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

package api

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLogServicePackError(t *testing.T) {
	tests := []struct {
		name      string
		ctx       func(t *testing.T) context.Context
		err       error
		wantLevel string
	}{
		{
			name:      "caller cancelled, no deadline",
			ctx:       cancelledContext(0),
			err:       fmt.Errorf("exit status 128"),
			wantLevel: "warn",
		},
		{
			name:      "wrapped cancel error",
			ctx:       func(*testing.T) context.Context { return context.Background() },
			err:       fmt.Errorf("git: %w", context.Canceled),
			wantLevel: "warn",
		},
		{
			name:      "caller cancelled with time left on the deadline",
			ctx:       cancelledContext(time.Hour),
			err:       fmt.Errorf("exit status 128"),
			wantLevel: "warn",
		},
		{
			// The operation deadline firing reaches us as a cancellation, and has to stay an error.
			name:      "cancelled as the deadline runs out",
			ctx:       cancelledContext(cancelDeadlineSlack / 2),
			err:       fmt.Errorf("exit status 128"),
			wantLevel: "error",
		},
		{
			name:      "unrelated failure",
			ctx:       func(*testing.T) context.Context { return context.Background() },
			err:       fmt.Errorf("exit status 128"),
			wantLevel: "error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			ctx := logger.WithContext(test.ctx(t))

			logServicePackError(ctx, test.err, enum.GitServiceTypeUploadPack, "/repo.git")

			require.Contains(t, buf.String(), fmt.Sprintf(`"level":%q`, test.wantLevel))
			require.Contains(t, buf.String(), "Fail to serve RPC(upload-pack) in /repo.git")
		})
	}
}

// cancelledContext returns a cancelled context, carrying a deadline deadlineIn away when that is non-zero. The
// cancellation still reports as context.Canceled, the shape a caller whose own deadline expired arrives in.
func cancelledContext(deadlineIn time.Duration) func(t *testing.T) context.Context {
	return func(t *testing.T) context.Context {
		t.Helper()

		ctx := context.Background()
		if deadlineIn != 0 {
			var cancelDeadline context.CancelFunc
			ctx, cancelDeadline = context.WithDeadline(ctx, time.Now().Add(deadlineIn))
			t.Cleanup(cancelDeadline)
		}

		ctx, cancel := context.WithCancel(ctx)
		cancel()
		require.ErrorIs(t, ctx.Err(), context.Canceled)

		return ctx
	}
}
