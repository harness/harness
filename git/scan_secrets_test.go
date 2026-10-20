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

package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitLogOptionsForSecretScanIncludesAllParents(t *testing.T) {
	tests := []struct {
		name    string
		baseRev string
		rev     string
		want    string
	}{
		{
			name: "full revision",
			rev:  "HEAD",
			want: "--no-merges HEAD",
		},
		{
			name:    "revision range",
			baseRev: "base",
			rev:     "HEAD",
			want:    "--no-merges base..HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitLogOptionsForSecretScan(tt.baseRev, tt.rev)
			require.Equal(t, tt.want, got)
			require.NotContains(t, got, "--first-parent")
		})
	}
}
