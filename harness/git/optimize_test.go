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

package git

import (
	"testing"
	"time"

	"github.com/harness/gitness/git/api"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/require"
)

func TestParseGCArgs(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	rfcDate := now.Format(api.RFC2822DateFormat)
	parsedDate, _ := time.Parse(api.RFC2822DateFormat, rfcDate)

	tests := []struct {
		name     string
		args     map[string]string
		expected api.GCParams
	}{
		{
			name: "All fields valid",
			args: map[string]string{
				"aggressive":        "true",
				"auto":              "false",
				"cruft":             "true",
				"max-cruft-size":    "123456",
				"prune":             rfcDate,
				"keep-largest-pack": "true",
			},
			expected: api.GCParams{
				Aggressive:      true,
				Auto:            false,
				Cruft:           ptr.Bool(true),
				MaxCruftSize:    123456,
				Prune:           parsedDate,
				KeepLargestPack: true,
			},
		},
		{
			name: "Prune as bool",
			args: map[string]string{
				"prune": "false",
			},
			expected: api.GCParams{
				Prune: ptr.Bool(false),
			},
		},
		{
			name: "Prune as fallback string",
			args: map[string]string{
				"prune": "2.weeks.ago",
			},
			expected: api.GCParams{
				Prune: "2.weeks.ago",
			},
		},
		{
			name: "Invalid bools ignored",
			args: map[string]string{
				"aggressive":        "yes",
				"keep-largest-pack": "maybe",
				"cruft":             "idk",
			},
			expected: api.GCParams{},
		},
		{
			name: "Invalid max-cruft-size ignored",
			args: map[string]string{
				"max-cruft-size": "not-a-number",
			},
			expected: api.GCParams{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGCArgs(tt.args)
			require.Equal(t, tt.expected.Aggressive, result.Aggressive)
			require.Equal(t, tt.expected.Auto, result.Auto)
			require.Equal(t, tt.expected.KeepLargestPack, result.KeepLargestPack)
			require.Equal(t, tt.expected.MaxCruftSize, result.MaxCruftSize)

			if tt.expected.Cruft != nil {
				require.NotNil(t, result.Cruft)
				require.Equal(t, *tt.expected.Cruft, *result.Cruft)
			} else {
				require.Nil(t, result.Cruft)
			}

			switch expected := tt.expected.Prune.(type) {
			case time.Time:
				actualTime, ok := result.Prune.(time.Time)
				require.True(t, ok)
				require.True(t, actualTime.Equal(expected))
			case *bool:
				actualBool, ok := result.Prune.(*bool)
				require.True(t, ok)
				require.Equal(t, *expected, *actualBool)
			case string:
				require.Equal(t, expected, result.Prune)
			case nil:
				require.Nil(t, result.Prune)
			default:
				t.Fatalf("unsupported prune type: %T", expected)
			}
		})
	}
}
