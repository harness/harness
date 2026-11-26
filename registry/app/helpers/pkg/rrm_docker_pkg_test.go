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

package pkg

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRrmDockerPackageTypeBasics(t *testing.T) {
	docker := NewDockerPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeDOCKER), docker.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeDocker), docker.GetPathPackageType())
	require.False(t, docker.IsFileOperationSupported())
}

func TestRrmDockerPackageTypeValidation(t *testing.T) {
	docker := NewDockerPackageType(nil)
	require.True(t, docker.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)))
	require.False(t, docker.IsValidRepoType("BAD"))
	require.True(t, docker.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceDockerhub)))
	require.False(t, docker.IsValidUpstreamSource("BAD"))
	require.False(t, docker.IsURLRequiredForUpstreamSource(string(artifact.UpstreamConfigSourceDockerhub)))
	require.True(t, docker.IsURLRequiredForUpstreamSource(string(artifact.UpstreamConfigSourceCustom)))
}

func TestRrmDockerPackageTypeURL(t *testing.T) {
	docker := NewDockerPackageType(nil)
	require.Empty(t, docker.GetPackageURL(context.Background(), "root", "reg"))
	require.Empty(t, docker.GetFilePath("img", "v1"))
}
