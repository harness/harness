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

package huggingface

import (
	"testing"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/stretchr/testify/require"
)

func TestRpfHuggingfaceArtifactInfo(t *testing.T) {
	info := ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{Image: "demo"},
		Repo:         "org/model",
		Revision:     "main",
		SHA256:       "abc123",
	}
	require.Equal(t, "demo", info.BaseArtifactInfo().Image)
	require.Equal(t, "main", info.GetVersion())
	require.Equal(t, "abc123", info.GetFileName())

	exists, value := info.GetImageVersion()
	require.True(t, exists)
	require.Equal(t, "org/model:main", value)

	exists, value = ArtifactInfo{Repo: "org/model"}.GetImageVersion()
	require.False(t, exists)
	require.Empty(t, value)

	exists, _ = ArtifactInfo{Revision: "main"}.GetImageVersion()
	require.False(t, exists)
}

func TestRpfNewPreUploadResponseFile(t *testing.T) {
	f := NewPreUploadResponseFile("weights.bin")
	require.Equal(t, "weights.bin", f.Path)
	require.Equal(t, LFSUpload, f.UploadMode)
	require.False(t, f.ShouldIgnore)
	require.NotEqual(t, RegularUpload, f.UploadMode)
}
