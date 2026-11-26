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
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRrmGenericPackageTypeBasics(t *testing.T) {
	generic := NewGenericPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeGENERIC), generic.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeGeneric), generic.GetPathPackageType())
	require.True(t, generic.IsFileOperationSupported())
	require.True(t, generic.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)))
}

func TestRrmNpmPackageTypeBasics(t *testing.T) {
	npm := NewNPMPackageType(nil, nil)
	require.Equal(t, string(artifact.PackageTypeNPM), npm.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeNpm), npm.GetPathPackageType())
	require.False(t, npm.IsFileOperationSupported())
	require.True(t, npm.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceNpmJs)))
}

func TestRrmMavenPackageTypeBasics(t *testing.T) {
	maven := NewMavenPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeMAVEN), maven.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeMaven), maven.GetPathPackageType())
	require.True(t, maven.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)))
}

func TestRrmGoPackageTypeBasics(t *testing.T) {
	gopkg := NewGoPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeGO), gopkg.GetPackageType())
	require.Equal(t, string(types.PathPackageTypeGo), gopkg.GetPathPackageType())
	require.False(t, gopkg.IsURLRequiredForUpstreamSource(string(artifact.UpstreamConfigSourceGoProxy)))
}

func TestRrmNugetPackageTypeBasics(t *testing.T) {
	nuget := NewNugetPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeNUGET), nuget.GetPackageType())
	require.True(t, nuget.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceNugetOrg)))
}

func TestRrmPythonPackageTypeBasics(t *testing.T) {
	python := NewPythonPackageType(nil)
	require.Equal(t, string(artifact.PackageTypePYTHON), python.GetPackageType())
	require.True(t, python.IsValidUpstreamSource(string(artifact.UpstreamConfigSourcePyPi)))
}

func TestRrmHelmPackageTypeBasics(t *testing.T) {
	helm := NewHelmPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeHELM), helm.GetPackageType())
	require.True(t, helm.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)))
}

func TestRrmRpmPackageTypeBasics(t *testing.T) {
	rpm := NewRPMPackageType(nil)
	require.Equal(t, string(artifact.PackageTypeRPM), rpm.GetPackageType())
	require.False(t, rpm.IsFileOperationSupported())
}

func TestRrmHuggingfacePackageTypeBasics(t *testing.T) {
	hf := NewHuggingFacePackageType(nil)
	require.Equal(t, string(artifact.PackageTypeHUGGINGFACE), hf.GetPackageType())
	require.False(t, hf.IsFileOperationSupported())
}
