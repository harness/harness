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

package limiter

import (
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCxrUnlimitedLimiter(t *testing.T) {
	ctx := t.Context()
	l := NewResourceLimiter()
	require.NoError(t, l.RepoCount(ctx, 1, 100))
	require.NoError(t, l.RepoSize(ctx, 1))
	require.NoError(t, l.RootSpaceStorage(ctx, 1))

	g := NewUnlimitedUsage()
	require.NoError(t, g.Usage(ctx, 1, enum.InfraProviderTypeDocker))
}

func TestCxrQuotaErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"repo soft", &RepoQuotaStorageError{Level: RepoQuotaLevelSoft, Size: 5}, "soft limit reached: SOFT, size: 5"},
		{"repo hard", &RepoQuotaStorageError{Level: RepoQuotaLevelHard, Size: 6}, "hard limit reached: HARD, size: 6"},
		{"repo over", &RepoQuotaStorageError{Level: RepoQuotaLevelOverLimit, Size: 7}, "over limit: OVER_LIMIT, size: 7"},
		{"repo unknown", &RepoQuotaStorageError{Level: "X"}, "unknown repo quota storage level: X"},
		{"total warn", &TotalQuotaStorageError{Level: TotalQuotaLevelWarn, Size: 1}, "warn limit reached: WARN, size: 1"},
		{
			"total critical", &TotalQuotaStorageError{Level: TotalQuotaLevelCritical, Size: 2},
			"critical limit reached: CRITICAL, size: 2",
		},
		{"total over", &TotalQuotaStorageError{Level: TotalQuotaLevelOverLimit, Size: 3}, "over limit: OVER_LIMIT, size: 3"},
		{"total unknown", &TotalQuotaStorageError{Level: "Y"}, "unknown total quota storage level: Y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Contains(t, tt.err.Error(), tt.want)
		})
	}
}

func TestCxrTotalQuotaHelpers(t *testing.T) {
	require.True(t, (&TotalQuotaStorageError{Level: TotalQuotaLevelOverLimit}).Blocked())
	require.False(t, (&TotalQuotaStorageError{Level: TotalQuotaLevelCritical}).Blocked())
	require.False(t, (&TotalQuotaStorageError{Level: TotalQuotaLevelWarn}).Blocked())

	msg := (&TotalQuotaStorageError{Size: 2 * 1024 * 1024, Limit: 1024 * 1024}).UserMessage()
	require.Equal(t, "Storage limit exceeded: 2.0 MiB of 1.0 MiB used.", msg)

	require.Equal(t, "0 B", FormatQuotaBytes(-5))
	require.Equal(t, "1.0 KiB", FormatQuotaBytes(1024))
}
