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

package limiter_test

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnlimited(t *testing.T) {
	ctx := context.Background()
	l := limiter.NewResourceLimiter()

	require.NoError(t, l.RepoCount(ctx, 1, 100))
	require.NoError(t, l.RepoSize(ctx, 1))
	require.NoError(t, l.RootSpaceStorage(ctx, 1, 1<<40))
}

// TestErrorsAreDistinct guards against a copy/paste mistake in the sentinels:
// callers use errors.Is to decide between blocking a push and only warning, so
// the values must not alias each other.
func TestErrorsAreDistinct(t *testing.T) {
	errs := []error{
		limiter.ErrMaxNumReposReached,
		limiter.ErrMaxRepoSizeReached,
		limiter.ErrMaxTotalStorageReached,
		limiter.ErrRepoSizeSoftLimitReached,
	}

	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				continue
			}
			assert.NotErrorIs(t, a, b, "%q must not match %q", a, b)
		}
	}
}
