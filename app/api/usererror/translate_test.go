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

package usererror

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"

	"github.com/stretchr/testify/assert"
)

// TestTranslate_TotalQuotaStorageError pins how the two verdicts a caller can bind
// from RootSpaceStorage reach the API response: only the enforced one is reported to
// the user, since the advisory levels are meant for the push output and must never
// have reached here in the first place.
func TestTranslate_TotalQuotaStorageError(t *testing.T) {
	ctx := context.Background()

	t.Run("over limit is forbidden with the user message", func(t *testing.T) {
		quotaErr := &limiter.TotalQuotaStorageError{
			Level: limiter.TotalQuotaLevelOverLimit,
			Size:  62 * 1024 * 1024,
			Limit: 50 * 1024 * 1024,
		}

		result := Translate(ctx, quotaErr)

		assert.Equal(t, http.StatusForbidden, result.Status)
		assert.Equal(t, quotaErr.UserMessage(), result.Message)
	})

	t.Run("wrapped over limit is still recognised", func(t *testing.T) {
		quotaErr := &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit}
		wrapped := fmt.Errorf("space 7: %w", quotaErr)

		result := Translate(ctx, wrapped)

		assert.Equal(t, http.StatusForbidden, result.Status)
		assert.Equal(t, quotaErr.UserMessage(), result.Message)
	})

	// An advisory-level verdict has nowhere it should legitimately reach Translate from -
	// every caller that can bind it filters it out - so one arriving here is a bug. It
	// must fall through to an internal error rather than be misreported as the limit
	// that stopped the request.
	t.Run("advisory level falls through to internal error", func(t *testing.T) {
		for _, level := range []limiter.TotalQuotaStorageLevel{
			limiter.TotalQuotaLevelWarn,
			limiter.TotalQuotaLevelCritical,
		} {
			t.Run(string(level), func(t *testing.T) {
				result := Translate(ctx, &limiter.TotalQuotaStorageError{Level: level})

				assert.Equal(t, http.StatusInternalServerError, result.Status)
			})
		}
	})
}

// TestTranslate_ErrMaxNumReposReached pins the sibling limiter rejection - a repo-count
// limit - to the same forbidden-with-message shape as the storage one above.
func TestTranslate_ErrMaxNumReposReached(t *testing.T) {
	result := Translate(context.Background(), limiter.ErrMaxNumReposReached)

	assert.Equal(t, http.StatusForbidden, result.Status)
	assert.Equal(t, limiter.ErrMaxNumReposReached.Error(), result.Message)
}

func TestTranslate_Timeouts(t *testing.T) {
	ctx := context.Background()

	t.Run("http client timeout is service unavailable", func(t *testing.T) {
		err := fmt.Errorf("access check failed: %w", &url.Error{Op: "Post", URL: "http://acl", Err: context.DeadlineExceeded})

		assert.Equal(t, http.StatusServiceUnavailable, Translate(ctx, err).Status)
	})

	t.Run("bare deadline stays internal", func(t *testing.T) {
		err := fmt.Errorf("work: %w", context.DeadlineExceeded)

		assert.Equal(t, http.StatusInternalServerError, Translate(ctx, err).Status)
	})

	t.Run("cancellation is still a closed request", func(t *testing.T) {
		err := fmt.Errorf("acl: %w", context.Canceled)

		assert.Equal(t, StatusClientClosedRequest, Translate(ctx, err).Status)
	})
}
