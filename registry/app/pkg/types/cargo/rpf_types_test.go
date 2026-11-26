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

package cargo

import (
	"testing"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/stretchr/testify/require"
)

func TestRpfCargoArtifactInfo(t *testing.T) {
	info := ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{Image: "demo"},
		Version:      "1.0.0",
		FileName:     "file.bin",
	}
	require.Equal(t, "demo", info.BaseArtifactInfo().Image)
	require.Equal(t, "1.0.0", info.GetVersion())
	require.Equal(t, "file.bin", info.GetFileName())
	exists, iv := info.GetImageVersion()
	require.True(t, exists)
	require.Equal(t, "demo:1.0.0", iv)

	exists, iv = ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}.GetImageVersion()
	require.False(t, exists)
	require.Empty(t, iv)

	exists, _ = ArtifactInfo{Version: "1.0.0"}.GetImageVersion()
	require.False(t, exists)
}
