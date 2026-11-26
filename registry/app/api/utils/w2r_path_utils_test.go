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

package utils

import (
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/stretchr/testify/require"
)

func TestW2rFilePathBuilders(t *testing.T) {
	require.Equal(t, "/org/example/lib/1.0", GetMavenFilePath("org.example:lib", "1.0"))
	require.Equal(t, "/org/example/lib", GetMavenFilePath("org.example:lib", ""))
	require.Equal(t, "/", GetMavenFilePath("no-colon", ""))

	require.Equal(t, "/pkg/1.2.3", GetGenericFilePath("pkg", "1.2.3"))
	require.Equal(t, "/pkg", GetGenericFilePath("pkg", ""))

	require.Equal(t, "/crates/serde/1.0.0", GetCargoFilePath("serde", "1.0.0"))
	require.Equal(t, "/crates/serde", GetCargoFilePath("serde", ""))

	require.Equal(t, "/mod/@v/v1.0.0", GetGoFilePath("mod", "v1.0.0"))
	require.Equal(t, "/mod/@v", GetGoFilePath("mod", ""))

	require.Equal(t, "/pkg/1.0-1/x86_64", GetRpmFilePath("pkg", "1.0-1.x86_64"))
	require.Equal(t, "/pkg/1.0-1/x86_64/2", GetRpmFilePath("pkg", "2:1.0-1.x86_64"))

	model := artifact.ArtifactType("model")
	require.Equal(t, "/model/org/repo/main", GetHuggingFaceFilePath("org/repo", &model, "main"))
	require.Equal(t, "/org/repo/main", GetHuggingFaceFilePath("org/repo", nil, "main"))
}

func TestW2rGetFilePathPerPackageType(t *testing.T) {
	tests := []struct {
		pkg  artifact.PackageType
		want string
		fail bool
	}{
		{artifact.PackageTypeNPM, "/n/1", false},
		{artifact.PackageTypePYTHON, "/n/1", false},
		{artifact.PackageTypeGENERIC, "/n/1", false},
		{artifact.PackageTypeNUGET, "/n/1", false},
		{artifact.PackageTypeCARGO, "/crates/n/1", false},
		{artifact.PackageTypeGO, "/n/@v/1", false},
		{artifact.PackageTypeDOCKER, "", true},
		{artifact.PackageTypeHELM, "", true},
		{artifact.PackageType("UNKNOWN"), "", true},
	}
	for _, tt := range tests {
		got, err := GetFilePath(tt.pkg, "n", "1")
		if tt.fail {
			require.Error(t, err, tt.pkg)
			continue
		}
		require.NoError(t, err, tt.pkg)
		require.Equal(t, tt.want, got, tt.pkg)
	}

	maven, err := GetFilePath(artifact.PackageTypeMAVEN, "g.r:n", "1")
	require.NoError(t, err)
	require.Equal(t, "/g/r/n/1", maven)

	rpm, err := GetFilePath(artifact.PackageTypeRPM, "n", "1.0.x86_64")
	require.NoError(t, err)
	require.Equal(t, "/n/1.0/x86_64", rpm)
}

func TestW2rGetFilePathWithArtifactType(t *testing.T) {
	model := artifact.ArtifactType("model")
	got, err := GetFilePathWithArtifactType(artifact.PackageTypeHUGGINGFACE, "repo", "main", &model)
	require.NoError(t, err)
	require.Equal(t, "/model/repo/main", got)

	_, err = GetFilePathWithArtifactType(artifact.PackageTypeNPM, "repo", "main", nil)
	require.Error(t, err)
}
