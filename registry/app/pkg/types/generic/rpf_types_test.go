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

package generic

import (
	"testing"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/stretchr/testify/require"
)

func TestRpfGenericArtifactInfo(t *testing.T) {
	info := ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{Image: "demo"},
		Version:      "1.0.0",
		FileName:     "file.bin",
		FilePath:     "/a/b/file.bin",
	}
	require.Equal(t, "demo", info.BaseArtifactInfo().Image)
	require.Equal(t, "1.0.0", info.GetVersion())
	require.Equal(t, "file.bin", info.GetFileName())

	tests := []struct {
		name       string
		info       ArtifactInfo
		wantExists bool
		wantValue  string
	}{
		{"image and version", info, true, "demo:1.0.0"},
		{"neither image nor version uses file path", ArtifactInfo{FilePath: "/x/y"}, true, "/x/y"},
		{"image only", ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}, false, ""},
		{"version only", ArtifactInfo{Version: "1"}, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exists, value := tc.info.GetImageVersion()
			require.Equal(t, tc.wantExists, exists)
			require.Equal(t, tc.wantValue, value)
		})
	}
}
