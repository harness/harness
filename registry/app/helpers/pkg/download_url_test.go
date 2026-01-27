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
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

// mockRegistryHelper is a mock implementation of interfaces.RegistryHelper.
type mockRegistryHelper struct{}

func (m *mockRegistryHelper) GetAuthHeaderPrefix() string {
	return "Bearer"
}

func (m *mockRegistryHelper) DeleteFileNode(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
) error {
	return nil
}

func (m *mockRegistryHelper) DeleteVersion(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ *types.Image,
	_ string,
	_ string,
	_ string,
) error {
	return nil
}

func (m *mockRegistryHelper) DeleteGenericImage(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
	_ string,
) error {
	return nil
}

func (m *mockRegistryHelper) ReportDeleteVersionEvent(
	_ context.Context,
	_ *registryevents.ArtifactDeletedPayload,
) {
}

func (m *mockRegistryHelper) ReportBuildPackageIndexEvent(
	_ context.Context,
	_ int64,
	_ string,
) {
}

func (m *mockRegistryHelper) ReportBuildRegistryIndexEvent(
	_ context.Context,
	_ int64,
	_ []types.SourceRef,
) {
}

func (m *mockRegistryHelper) GetPackageURL(
	_ context.Context,
	rootIdentifier string,
	registryIdentifier string,
	packageTypePathParam string,
) string {
	return "https://registry.example.com/" + rootIdentifier + "/" + registryIdentifier + "/" + packageTypePathParam
}

func (m *mockRegistryHelper) GetHostName(_ context.Context, _ string) string {
	return "registry.example.com"
}

func (m *mockRegistryHelper) GetArtifactMetadata(
	_ types.ArtifactMetadata,
	_ string,
) *artifact.ArtifactMetadata {
	return nil
}

func (m *mockRegistryHelper) GetArtifactVersionMetadata(
	_ types.NonOCIArtifactMetadata,
	_ string,
	_ string,
) *artifact.ArtifactVersionMetadata {
	return nil
}

func (m *mockRegistryHelper) GetFileMetadata(
	_ types.FileNodeMetadata,
	_ string,
	_ string,
) *artifact.FileDetail {
	return nil
}

func (m *mockRegistryHelper) GetArtifactDetail(
	_ *types.Image,
	_ *types.Artifact,
	_ map[string]any,
	_ int64,
) *artifact.ArtifactDetail {
	return nil
}

func (m *mockRegistryHelper) ReplacePlaceholders(
	_ context.Context,
	_ *[]artifact.ClientSetupSection,
	_ string,
	_ string,
	_ *artifact.ArtifactParam,
	_ *artifact.VersionParam,
	_ string,
	_ string,
	_ string,
	_ string,
) {
}

func TestNugetPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	nugetPackage := NewNugetPackageType(mockHelper)

	tests := []struct {
		name               string
		rootIdentifier     string
		registryIdentifier string
		packageName        string
		artifactType       string
		version            string
		filename           string
		filepath           string
		expectedURL        string
		expectError        bool
	}{
		{
			name:               "simple nuget package",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "Newtonsoft.Json",
			artifactType:       "",
			version:            "13.0.1",
			filename:           "newtonsoft.json.13.0.1.nupkg",
			filepath:           "/Newtonsoft.Json/13.0.1/newtonsoft.json.13.0.1.nupkg",
			expectedURL: "https://registry.example.com/root1/registry1/nuget/package/" +
				"Newtonsoft.Json/13.0.1/newtonsoft.json.13.0.1.nupkg",
			expectError: false,
		},
		{
			name:               "nuget package with complex name",
			rootIdentifier:     "myroot",
			registryIdentifier: "myreg",
			packageName:        "Microsoft.Extensions.Logging",
			artifactType:       "",
			version:            "7.0.0",
			filename:           "microsoft.extensions.logging.7.0.0.nupkg",
			filepath:           "/Microsoft.Extensions.Logging/7.0.0/microsoft.extensions.logging.7.0.0.nupkg",
			expectedURL: "https://registry.example.com/myroot/myreg/nuget/package/" +
				"Microsoft.Extensions.Logging/7.0.0/microsoft.extensions.logging.7.0.0.nupkg",
			expectError: false,
		},
		{
			name:               "empty packageName",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "",
			artifactType:       "",
			version:            "13.0.1",
			filename:           "test.nupkg",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
		{
			name:               "empty version",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "Newtonsoft.Json",
			artifactType:       "",
			version:            "",
			filename:           "test.nupkg",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
		{
			name:               "empty filename",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "Newtonsoft.Json",
			artifactType:       "",
			version:            "13.0.1",
			filename:           "",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := nugetPackage.GetPkgDownloadURL(
				context.Background(),
				tt.rootIdentifier,
				tt.registryIdentifier,
				tt.packageName,
				tt.artifactType,
				tt.version,
				tt.filename,
				tt.filepath,
			)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be empty")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

func TestNPMPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	npmPackage := NewNPMPackageType(mockHelper)

	tests := []struct {
		name               string
		rootIdentifier     string
		registryIdentifier string
		packageName        string
		artifactType       string
		version            string
		filename           string
		filepath           string
		expectedURL        string
		expectError        bool
	}{
		{
			name:               "unscoped npm package",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "express",
			artifactType:       "",
			version:            "4.18.2",
			filename:           "express-4.18.2.tgz",
			filepath:           "/express/4.18.2/express-4.18.2.tgz",
			expectedURL:        "https://registry.example.com/root1/registry1/npm/express/-/4.18.2/express-4.18.2.tgz",
			expectError:        false,
		},
		{
			name:               "scoped npm package",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "@angular/core",
			artifactType:       "",
			version:            "15.0.0",
			filename:           "core-15.0.0.tgz",
			filepath:           "/@angular/core/15.0.0/core-15.0.0.tgz",
			//nolint:lll
			expectedURL: "https://registry.example.com/root1/registry1/npm/@angular/core/-/15.0.0/core-15.0.0.tgz",
			expectError: false,
		},
		{
			name:               "scoped package with organization",
			rootIdentifier:     "myroot",
			registryIdentifier: "myreg",
			packageName:        "@babel/preset-env",
			artifactType:       "",
			version:            "7.20.0",
			filename:           "preset-env-7.20.0.tgz",
			filepath:           "/@babel/preset-env/7.20.0/preset-env-7.20.0.tgz",
			//nolint:lll
			expectedURL: "https://registry.example.com/myroot/myreg/npm/@babel/preset-env/-/7.20.0/preset-env-7.20.0.tgz",
			expectError: false,
		},
		{
			name:               "empty packageName",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "",
			artifactType:       "",
			version:            "4.18.2",
			filename:           "test.tgz",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
		{
			name:               "empty version",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "express",
			artifactType:       "",
			version:            "",
			filename:           "test.tgz",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
		{
			name:               "empty filename",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			packageName:        "express",
			artifactType:       "",
			version:            "4.18.2",
			filename:           "",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := npmPackage.GetPkgDownloadURL(
				context.Background(),
				tt.rootIdentifier,
				tt.registryIdentifier,
				tt.packageName,
				tt.artifactType,
				tt.version,
				tt.filename,
				tt.filepath,
			)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be empty")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

func TestMavenPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	mavenPackage := NewMavenPackageType(mockHelper)

	tests := []struct {
		name               string
		rootIdentifier     string
		registryIdentifier string
		filepath           string
		expectedURL        string
		expectError        bool
	}{
		{
			name:               "simple maven artifact",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			filepath:           "/com/example/my-artifact/1.0.0/my-artifact-1.0.0.jar",
			expectedURL: "https://registry.example.com/root1/registry1/maven/" +
				"com/example/my-artifact/1.0.0/my-artifact-1.0.0.jar",
			expectError: false,
		},
		{
			name:               "spring boot artifact",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			filepath:           "/org/springframework/boot/spring-boot-starter/2.7.0/spring-boot-starter-2.7.0.jar",
			expectedURL: "https://registry.example.com/root1/registry1/maven/" +
				"org/springframework/boot/spring-boot-starter/2.7.0/spring-boot-starter-2.7.0.jar",
			expectError: false,
		},
		{
			name:               "maven artifact with pom file",
			rootIdentifier:     "myroot",
			registryIdentifier: "myreg",
			filepath:           "/com/google/guava/guava/31.1-jre/guava-31.1-jre.pom",
			expectedURL: "https://registry.example.com/myroot/myreg/maven/" +
				"com/google/guava/guava/31.1-jre/guava-31.1-jre.pom",
			expectError: false,
		},
		{
			name:               "empty filepath",
			rootIdentifier:     "root1",
			registryIdentifier: "registry1",
			filepath:           "",
			expectedURL:        "",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := mavenPackage.GetPkgDownloadURL(
				context.Background(),
				tt.rootIdentifier,
				tt.registryIdentifier,
				"",
				"",
				"",
				"",
				tt.filepath,
			)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be empty")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

func TestPythonPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	pythonPackage := NewPythonPackageType(mockHelper)

	url, err := pythonPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"requests",
		"",
		"2.28.0",
		"requests-2.28.0.tar.gz",
		"/requests/2.28.0/requests-2.28.0.tar.gz",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestRPMPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	rpmPackage := NewRPMPackageType(mockHelper)

	url, err := rpmPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"nginx",
		"x86_64",
		"1.20.1.x86_64",
		"nginx-1.20.1.x86_64.rpm",
		"/nginx/1.20.1.x86_64/nginx-1.20.1.x86_64.rpm",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestCargoPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	cargoPackage := NewCargoPackageType(mockHelper, nil)

	url, err := cargoPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"serde",
		"",
		"1.0.0",
		"serde-1.0.0.crate",
		"/serde/1.0.0/serde-1.0.0.crate",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestGoPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	goPackage := NewGoPackageType(mockHelper)

	url, err := goPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"github.com/gin-gonic/gin",
		"",
		"v1.9.0",
		"gin-v1.9.0.zip",
		"/github.com/gin-gonic/gin/v1.9.0/gin-v1.9.0.zip",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestGenericPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	genericPackage := NewGenericPackageType(mockHelper)

	tests := []struct {
		name        string
		packageName string
		version     string
		filename    string
		filepath    string
	}{
		{
			name:        "simple generic artifact",
			packageName: "myfile",
			version:     "1.0.0",
			filename:    "myfile-1.0.0.bin",
			filepath:    "/myfile/1.0.0/myfile-1.0.0.bin",
		},
		{
			name:        "generic artifact with complex name",
			packageName: "my-package",
			version:     "2.5.3",
			filename:    "my-package-2.5.3.tar.gz",
			filepath:    "/my-package/2.5.3/my-package-2.5.3.tar.gz",
		},
		{
			name:        "generic artifact with special characters in filename",
			packageName: "artifact",
			version:     "1.0.0-beta",
			filename:    "artifact_v1.0.0-beta.zip",
			filepath:    "/artifact/1.0.0-beta/artifact_v1.0.0-beta.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := genericPackage.GetPkgDownloadURL(
				context.Background(),
				"root1",
				"registry1",
				tt.packageName,
				"",
				tt.version,
				tt.filename,
				tt.filepath,
			)
			assert.NoError(t, err)
			assert.Equal(t, "", url)
		})
	}
}

func TestHuggingFacePackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	hfPackage := NewHuggingFacePackageType(mockHelper)

	url, err := hfPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"bert-base-uncased",
		"model",
		"v1.0",
		"pytorch_model.bin",
		"/bert-base-uncased/v1.0/pytorch_model.bin",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestDockerPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	dockerPackage := NewDockerPackageType(mockHelper)

	url, err := dockerPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"nginx",
		"",
		"sha256:abcd1234",
		"manifest.json",
		"/nginx/sha256:abcd1234/manifest.json",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}

func TestHelmPackageType_GetPkgDownloadURL(t *testing.T) {
	mockHelper := &mockRegistryHelper{}
	helmPackage := NewHelmPackageType(mockHelper)

	url, err := helmPackage.GetPkgDownloadURL(
		context.Background(),
		"root1",
		"registry1",
		"mychart",
		"",
		"sha256:efgh5678",
		"chart.yaml",
		"/mychart/sha256:efgh5678/chart.yaml",
	)
	assert.NoError(t, err)
	assert.Equal(t, "", url)
}
