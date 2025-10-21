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

package contextutil

import (
	"context"
	"testing"
	"time"
)

func TestWithNewTimeout(t *testing.T) {
	t.Run("creates new context with timeout", func(t *testing.T) {
		ctx := context.Background()
		timeout := 100 * time.Millisecond

		newCtx, cancel := WithNewTimeout(ctx, timeout)
		defer cancel()

		if newCtx == nil {
			t.Fatal("expected non-nil context")
		}

		deadline, ok := newCtx.Deadline()
		if !ok {
			t.Fatal("expected context to have deadline")
		}

		expectedDeadline := time.Now().Add(timeout)
		if deadline.After(expectedDeadline.Add(50 * time.Millisecond)) {
			t.Errorf("deadline is too far in the future: got %v, expected around %v", deadline, expectedDeadline)
		}
	})

	t.Run("new context is not canceled when parent is canceled", func(t *testing.T) {
		parentCtx, parentCancel := context.WithCancel(context.Background())
		timeout := 1 * time.Second

		newCtx, cancel := WithNewTimeout(parentCtx, timeout)
		defer cancel()

		// Cancel parent context
		parentCancel()

		// Give it a moment to propagate
		time.Sleep(10 * time.Millisecond)

		// New context should not be canceled
		select {
		case <-newCtx.Done():
			t.Fatal("new context should not be canceled when parent is canceled")
		default:
			// Expected: context is not canceled
		}
	})

	t.Run("new context times out after specified duration", func(t *testing.T) {
		ctx := context.Background()
		timeout := 50 * time.Millisecond

		newCtx, cancel := WithNewTimeout(ctx, timeout)
		defer cancel()

		select {
		case <-newCtx.Done():
			t.Fatal("context should not be done immediately")
		case <-time.After(10 * time.Millisecond):
			// Expected: context is not done yet
		}

		// Wait for timeout
		select {
		case <-newCtx.Done():
			// Expected: context is done after timeout
			if newCtx.Err() != context.DeadlineExceeded {
				t.Errorf("expected DeadlineExceeded error, got %v", newCtx.Err())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should have timed out")
		}
	})

	t.Run("cancel function works correctly", func(t *testing.T) {
		ctx := context.Background()
		timeout := 1 * time.Second

		newCtx, cancel := WithNewTimeout(ctx, timeout)

		// Cancel immediately
		cancel()

		select {
		case <-newCtx.Done():
			// Expected: context is canceled
			if newCtx.Err() != context.Canceled {
				t.Errorf("expected Canceled error, got %v", newCtx.Err())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should have been canceled")
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		ctx := context.Background()
		timeout := 0 * time.Second

		newCtx, cancel := WithNewTimeout(ctx, timeout)
		defer cancel()

		// Context with zero timeout should be immediately done
		select {
		case <-newCtx.Done():
			// Expected: context is done
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context with zero timeout should be immediately done")
		}
	})

	t.Run("negative timeout", func(t *testing.T) {
		ctx := context.Background()
		timeout := -1 * time.Second

		newCtx, cancel := WithNewTimeout(ctx, timeout)
		defer cancel()

		// Context with negative timeout should be immediately done
		select {
		case <-newCtx.Done():
			// Expected: context is done
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context with negative timeout should be immediately done")
		}
	})
}
