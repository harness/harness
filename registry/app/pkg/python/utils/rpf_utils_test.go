//  Copyright 2023 Harness, Inc.
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

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRpfIsMainArtifactFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"demo-1.0-py3-none-any.whl", true},
		{"demo-1.0.tar.gz", true},
		{"DEMO-1.0.TAR.GZ", true},
		{"demo-1.0.WHL", true},
		{"demo-1.0.zip", false},
		{"demo-1.0.whl.asc", false},
		{"demo-1.0.tar.gz.sha256", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsMainArtifactFile(tc.name))
		})
	}
}
