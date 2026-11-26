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

package helpers

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/factory"
	helperpkg "github.com/harness/gitness/registry/app/helpers/pkg"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func rrmSetupPackageWrapper(t *testing.T) *packageWrapper {
	t.Helper()
	pf := factory.NewPackageFactory()
	pf.Register(helperpkg.NewDockerPackageType(nil))
	pf.Register(helperpkg.NewCargoPackageType(nil, nil))
	w := NewPackageWrapper(pf, nil)
	pw, ok := w.(*packageWrapper)
	require.True(t, ok)
	return pw
}

func TestRrmPackageWrapperValidation(t *testing.T) {
	w := rrmSetupPackageWrapper(t)

	require.True(t, w.IsValidPackageType(""))
	require.True(t, w.IsValidPackageType(string(artifact.PackageTypeDOCKER)))
	require.False(t, w.IsValidPackageType("NOT_REAL"))

	require.True(t, w.IsValidPackageTypes([]string{string(artifact.PackageTypeDOCKER)}))
	require.False(t, w.IsValidPackageTypes([]string{"NOT_REAL"}))

	require.True(t, w.IsValidRepoType(""))
	require.True(t, w.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)))
	require.False(t, w.IsValidRepoType("NOT_REAL"))

	require.True(t, w.ValidateRepoType(string(artifact.PackageTypeDOCKER), string(artifact.RegistryTypeVIRTUAL)))
	require.False(t, w.ValidateRepoType(string(artifact.PackageTypeDOCKER), "BAD"))

	require.True(t, w.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceDockerhub)))
	require.False(t, w.ValidateUpstreamSource(string(artifact.PackageTypeDOCKER), "BAD"))
}

func TestRrmPackageWrapperGetPackageAndPaths(t *testing.T) {
	w := rrmSetupPackageWrapper(t)

	pkg := w.GetPackage(string(artifact.PackageTypeDOCKER))
	require.NotNil(t, pkg)
	require.False(t, w.IsFileOperationSupported(string(artifact.PackageTypeDOCKER)))
	require.False(t, w.IsFileOperationSupported(""))
	require.False(t, w.IsFileOperationSupported("unknown"))

	pt, err := w.GetPackageTypeFromPathPackageType(string(types.PathPackageTypeDocker))
	require.NoError(t, err)
	require.Equal(t, string(artifact.PackageTypeDOCKER), pt)

	_, err = w.GetPackageTypeFromPathPackageType("unknown-path")
	require.Error(t, err)

	pn, ver, at, err := w.GetPackageAndVersionFromNodePath(string(artifact.PackageTypeDOCKER), "/any")
	require.NoError(t, err)
	require.Empty(t, pn)
	require.Empty(t, ver)
	require.Empty(t, at)
}

func TestRrmPackageWrapperURLRequired(t *testing.T) {
	w := rrmSetupPackageWrapper(t)
	require.False(t, w.IsURLRequiredForUpstreamSource(
		string(artifact.PackageTypeDOCKER),
		string(artifact.UpstreamConfigSourceDockerhub),
	))
	require.True(t, w.IsURLRequiredForUpstreamSource(string(artifact.PackageTypeDOCKER), "unknown"))
}

func TestRrmPackageWrapperGettersNilPackage(t *testing.T) {
	w := rrmSetupPackageWrapper(t)
	require.Nil(t, w.GetArtifactMetadata(types.ArtifactMetadata{PackageType: "unknown"}))
	require.Nil(t, w.GetArtifactVersionMetadata("unknown", "img", types.NonOCIArtifactMetadata{}))
	require.Nil(t, w.GetFileMetadata(context.Background(), "r", "reg", "unknown", "a", "v", types.FileNodeMetadata{}))
}
