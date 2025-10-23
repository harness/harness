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

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/types"

	ocidigest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Test helpers ----

func toPtr[T any](v T) *T { return &v }

func ts(n int64) time.Time {
	// Produce a deterministic time; mapper uses GetTimeInMs so only monotonicity matters.
	return time.UnixMilli(n)
}

// Fake URL provider used in tests.
type fakeURLProvider struct {
	base string
}

func (f fakeURLProvider) GetInternalAPIURL(_ context.Context) string {
	return f.base + "/api"
}

func (f fakeURLProvider) GenerateUIRepoURL(_ context.Context, repoPath string) string {
	return f.base + "/ui/repo/" + repoPath
}

func (f fakeURLProvider) GenerateUIPRURL(_ context.Context, repoPath string, prID int64) string {
	return f.base + "/ui/repo/" + repoPath + "/pr/" + fmt.Sprintf("%d", prID)
}

func (f fakeURLProvider) GenerateUICompareURL(_ context.Context, repoPath string, ref1 string, ref2 string) string {
	return f.base + "/ui/repo/" + repoPath + "/compare/" + ref1 + "..." + ref2
}

func (f fakeURLProvider) GenerateUIRefURL(_ context.Context, repoPath string, ref string) string {
	return f.base + "/ui/repo/" + repoPath + "/ref/" + ref
}

func (f fakeURLProvider) GetAPIHostname(_ context.Context) string {
	return "fake.api.hostname"
}

func (f fakeURLProvider) GetGITHostname(_ context.Context) string {
	return "fake.git.hostname"
}

func (f fakeURLProvider) GetAPIProto(_ context.Context) string {
	return "https"
}

func (f fakeURLProvider) RegistryURL(_ context.Context, params ...string) string {
	if len(params) >= 2 {
		return f.base + "/" + params[0] + "/" + params[1]
	}
	return f.base + "/default"
}

func (f fakeURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return f.base + "/package/" + regRef + "/" + pkgType
}

func (f fakeURLProvider) GetUIBaseURL(_ context.Context, _ ...string) string {
	return f.base + "/ui"
}

func (f fakeURLProvider) GenerateUIRegistryURL(_ context.Context, parentSpacePath string, registryName string) string {
	return f.base + "/ui/registry/" + parentSpacePath + "/" + registryName
}

func (f fakeURLProvider) GenerateContainerGITCloneURL(_ context.Context, repo string) string {
	return f.base + "/container/" + repo
}

func (f fakeURLProvider) GenerateGITCloneSSHURL(_ context.Context, repo string) string {
	return "ssh://fake.registry/" + repo
}

func (f fakeURLProvider) GenerateGITCloneURL(_ context.Context, repo string) string {
	return f.base + "/" + repo
}

func (f fakeURLProvider) GenerateUIBuildURL(_ context.Context, repo string, _ string, _ int64) string {
	return f.base + "/ui/" + repo
}

func TestGetArtifactMetadata_WithGenericPackage(t *testing.T) {
	ctx := context.Background()
	p := &fakeURLProvider{base: "https://test.registry"}

	artifacts := []types.ArtifactMetadata{
		{
			RepoName:      "test-repo",
			Name:          "generic-artifact",
			Version:       "1.0.0",
			Labels:        []string{"env=prod"},
			ModifiedAt:    ts(1000),
			PackageType:   artifactapi.PackageTypeGENERIC,
			DownloadCount: 10,
			IsQuarantined: false,
		},
	}

	result := GetArtifactMetadata(ctx, artifacts, "root", p, "Bearer")

	require.Len(t, result, 1)
	require.Equal(t, "test-repo", result[0].RegistryIdentifier)
	require.Equal(t, "generic-artifact", result[0].Name)
	require.Equal(t, "1.0.0", *result[0].Version)
	require.Equal(t, artifactapi.PackageTypeGENERIC, *result[0].PackageType)
	require.EqualValues(t, 10, *result[0].DownloadsCount)
	require.EqualValues(t, false, *result[0].IsQuarantined)
}

func TestGetArtifactMetadata_WithDockerPackage(t *testing.T) {
	ctx := context.Background()
	p := &fakeURLProvider{base: "https://docker.registry"}

	artifacts := []types.ArtifactMetadata{
		{
			RepoName:         "docker-repo",
			Name:             "nginx",
			Version:          "latest",
			Labels:           []string{"web=server"},
			ModifiedAt:       ts(2000),
			PackageType:      artifactapi.PackageTypeDOCKER,
			DownloadCount:    50,
			IsQuarantined:    true,
			QuarantineReason: toPtr("security scan failed"),
		},
	}

	result := GetArtifactMetadata(ctx, artifacts, "root", p, "Token")

	require.Len(t, result, 1)
	require.Equal(t, "docker-repo", result[0].RegistryIdentifier)
	require.Equal(t, "nginx", result[0].Name)
	require.Equal(t, artifactapi.PackageTypeDOCKER, *result[0].PackageType)
	require.EqualValues(t, true, *result[0].IsQuarantined)
	require.Equal(t, "security scan failed", *result[0].QuarantineReason)
}

func TestGetRegistryArtifactMetadata_MultipleArtifacts(t *testing.T) {
	artifacts := []types.ArtifactMetadata{
		{
			RepoName:      "repo1",
			Name:          "artifact1",
			LatestVersion: "1.0.0",
			Labels:        []string{"label1"},
			ModifiedAt:    ts(1000),
			PackageType:   artifactapi.PackageTypeMAVEN,
			DownloadCount: 5,
			IsQuarantined: false,
		},
		{
			RepoName:      "repo2",
			Name:          "artifact2",
			LatestVersion: "2.0.0",
			Labels:        []string{"label2"},
			ModifiedAt:    ts(2000),
			PackageType:   artifactapi.PackageTypeNPM,
			DownloadCount: 15,
			IsQuarantined: true,
		},
	}

	result := GetRegistryArtifactMetadata(artifacts)

	require.Len(t, result, 2)
	require.Equal(t, "repo1", result[0].RegistryIdentifier)
	require.Equal(t, "artifact1", result[0].Name)
	require.Equal(t, "1.0.0", result[0].LatestVersion)
	require.Equal(t, artifactapi.PackageTypeMAVEN, *result[0].PackageType)

	require.Equal(t, "repo2", result[1].RegistryIdentifier)
	require.Equal(t, "artifact2", result[1].Name)
	require.Equal(t, "2.0.0", result[1].LatestVersion)
	require.Equal(t, artifactapi.PackageTypeNPM, *result[1].PackageType)
}

func TestGetMavenArtifactDetail_WithMultipleFiles(t *testing.T) {
	image := &types.Image{Name: "com.example:my-artifact"}
	artifact := &types.Artifact{
		Version:   "1.2.3",
		CreatedAt: ts(100),
		UpdatedAt: ts(200),
	}
	mavenMeta := metadata.MavenMetadata{
		Files: []metadata.File{
			{Size: 1024},
			{Size: 2048},
			{Size: 512},
		},
	}

	result := GetMavenArtifactDetail(image, artifact, mavenMeta)

	require.Equal(t, "com.example:my-artifact", *result.Name)
	require.Equal(t, "1.2.3", result.Version)
	require.Equal(t, "3.50KB", *result.Size) // 1024 + 2048 + 512 = 3584 bytes
	require.NotNil(t, result.CreatedAt)
	require.NotNil(t, result.ModifiedAt)
}

func TestGetMavenArtifactDetail_EmptyFiles(t *testing.T) {
	image := &types.Image{Name: "empty.artifact"}
	artifact := &types.Artifact{
		Version:   "0.0.1",
		CreatedAt: ts(50),
		UpdatedAt: ts(100),
	}
	mavenMeta := metadata.MavenMetadata{
		Files: []metadata.File{},
	}

	result := GetMavenArtifactDetail(image, artifact, mavenMeta)

	require.Equal(t, "empty.artifact", *result.Name)
	require.Equal(t, "0.0.1", result.Version)
	require.Equal(t, "0.00B", *result.Size)
}

func TestGetGenericArtifactDetail_WithDescription(t *testing.T) {
	image := &types.Image{Name: "generic-app"}
	artifact := &types.Artifact{
		Version:   "1.0.0",
		CreatedAt: ts(100),
		UpdatedAt: ts(200),
	}
	genericMeta := metadata.GenericMetadata{
		Description: "A generic application artifact",
	}

	result := GetGenericArtifactDetail(image, artifact, genericMeta)

	require.Equal(t, toPtr("generic-app"), result.Name)
	require.Equal(t, "1.0.0", result.Version)
	require.NotNil(t, result.CreatedAt)
	require.NotNil(t, result.ModifiedAt)
}

func TestGetPythonArtifactDetail_WithMetadata(t *testing.T) {
	image := &types.Image{Name: "python-package"}
	artifact := &types.Artifact{
		Version:   "2.1.0",
		CreatedAt: ts(150),
		UpdatedAt: ts(250),
	}
	pythonMeta := map[string]interface{}{
		"author":      "John Doe",
		"description": "A Python package",
		"requires":    "requests>=2.0",
	}

	result := GetPythonArtifactDetail(image, artifact, pythonMeta)

	require.Equal(t, toPtr("python-package"), result.Name)
	require.Equal(t, "2.1.0", result.Version)
	require.NotNil(t, result.CreatedAt)
	require.NotNil(t, result.ModifiedAt)
}

func TestGetNPMArtifactDetail_WithSize(t *testing.T) {
	image := &types.Image{Name: "npm-package"}
	artifact := &types.Artifact{
		Version:   "3.0.0",
		CreatedAt: ts(300),
		UpdatedAt: ts(400),
	}

	npmMeta := npm2.NpmMetadata{Size: 5120}
	metadataBytes, _ := json.Marshal(npmMeta)
	artifact.Metadata = metadataBytes

	extraMeta := map[string]interface{}{
		"dependencies": map[string]string{"lodash": "^4.0.0"},
	}

	result := GetNPMArtifactDetail(image, artifact, extraMeta, 25)

	require.Equal(t, toPtr("npm-package"), result.Name)
	require.Equal(t, "3.0.0", result.Version)
	require.Equal(t, toPtr("5120"), result.Size)
	require.EqualValues(t, 25, *result.DownloadCount)
}

func TestGetNugetArtifactDetail_WithMetadata(t *testing.T) {
	image := &types.Image{Name: "nuget-package"}
	artifact := &types.Artifact{
		Version:   "4.0.0",
		CreatedAt: ts(400),
		UpdatedAt: ts(500),
	}
	nugetMeta := map[string]interface{}{
		"size":        float64(8192),
		"description": "A .NET package",
		"authors":     "Microsoft",
	}

	result := GetNugetArtifactDetail(image, artifact, nugetMeta, 100)

	assert.Equal(t, "nuget-package", *result.Name)
	require.Equal(t, "4.0.0", result.Version)
	assert.Equal(t, "8.00KB", *result.Size)
	require.EqualValues(t, 100, *result.DownloadCount)
}

func TestGetCargoArtifactDetail_WithMetadata(t *testing.T) {
	image := &types.Image{Name: "cargo-crate"}
	artifact := &types.Artifact{
		Version:   "0.5.0",
		CreatedAt: ts(500),
		UpdatedAt: ts(600),
	}
	cargoMeta := map[string]interface{}{
		"size":        float64(4096),
		"description": "A Rust crate",
		"license":     "MIT",
	}

	result := GetCargoArtifactDetail(image, artifact, cargoMeta, 75)

	assert.Equal(t, "cargo-crate", *result.Name)
	require.Equal(t, "0.5.0", result.Version)
	assert.Equal(t, "4.00KB", *result.Size)
	require.EqualValues(t, 75, *result.DownloadCount)
}

func TestGetGoArtifactDetail_WithMetadata(t *testing.T) {
	image := &types.Image{Name: "go-module"}
	artifact := &types.Artifact{
		Version:   "1.18.0",
		CreatedAt: ts(600),
		UpdatedAt: ts(700),
	}
	goMeta := map[string]interface{}{
		"size":   float64(2048),
		"module": "github.com/example/go-module",
	}

	result := GetGoArtifactDetail(image, artifact, goMeta, 200)

	assert.Equal(t, "go-module", *result.Name)
	require.Equal(t, "1.18.0", result.Version)
	assert.Equal(t, "2.00KB", *result.Size)
	require.EqualValues(t, 200, *result.DownloadCount)
}

func TestGetHFArtifactDetail_WithMetadata(t *testing.T) {
	image := &types.Image{
		Name:         "hf-model",
		ArtifactType: toPtr(artifactapi.ArtifactTypeModel),
	}
	artifact := &types.Artifact{
		Version:   "2.1.0",
		CreatedAt: ts(700),
		UpdatedAt: ts(800),
	}
	hfMeta := map[string]interface{}{
		"size":        float64(1073741824), // 1GB
		"model_type":  "transformer",
		"description": "A Hugging Face model",
	}

	result := GetHFArtifactDetail(image, artifact, hfMeta, 150)

	assert.Equal(t, "hf-model", *result.Name)
	require.Equal(t, "2.1.0", result.Version)
	assert.Equal(t, "1.00GB", *result.Size)
	require.EqualValues(t, 150, *result.DownloadCount)
}

func TestGetDockerArtifactDetails_Complete(t *testing.T) {
	registry := &types.Registry{
		Name:        "docker-registry",
		PackageType: artifactapi.PackageTypeDOCKER,
	}
	tag := &types.TagDetail{
		ImageName: "nginx",
		Name:      "1.21",
		CreatedAt: ts(1000),
		UpdatedAt: ts(1100),
	}
	manifest := &types.Manifest{
		TotalSize: 134217728, // 128MB
	}
	manifest.Digest = ocidigest.NewDigestFromHex("sha256", "abcdef123456")

	result := GetDockerArtifactDetails(registry, tag, manifest,
		"https://registry.example.com", 200, true, toPtr("vulnerability found"))

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Equal(t, "nginx", result.Data.ImageName)
	require.Equal(t, "1.21", result.Data.Version)
	require.Equal(t, artifactapi.PackageTypeDOCKER, result.Data.PackageType)
	assert.Equal(t, "128.00MB", *result.Data.Size)
	require.EqualValues(t, 200, *result.Data.DownloadsCount)
	require.EqualValues(t, true, *result.Data.IsQuarantined)
	require.Equal(t, "vulnerability found", *result.Data.QuarantineReason)
	require.Contains(t, result.Data.RegistryPath, "docker-registry/nginx/sha256:abcdef123456")
}

func TestGetHelmArtifactDetails_Complete(t *testing.T) {
	registry := &types.Registry{
		Name:        "helm-registry",
		PackageType: artifactapi.PackageTypeHELM,
	}
	tag := &types.TagDetail{
		ImageName: "wordpress",
		Name:      "15.2.0",
	}
	manifest := &types.Manifest{
		TotalSize: 65536, // 64KB
	}
	manifest.Digest = ocidigest.NewDigestFromHex("sha256", "fedcba654321")

	result := GetHelmArtifactDetails(registry, tag, manifest, "https://helm.example.com", 150)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	assert.Equal(t, "wordpress", *result.Data.Artifact)
	require.Equal(t, "15.2.0", result.Data.Version)
	require.Equal(t, artifactapi.PackageTypeHELM, result.Data.PackageType)
	assert.Equal(t, "64.00KB", *result.Data.Size)
	require.EqualValues(t, 150, *result.Data.DownloadsCount)
	require.Contains(t, result.Data.RegistryPath, "helm-registry/wordpress/sha256:fedcba654321")
}

func TestGetArtifactSummary_Complete(t *testing.T) {
	artifact := types.ImageMetadata{
		Name:          "test-image",
		PackageType:   artifactapi.PackageTypeDOCKER,
		ArtifactType:  toPtr(artifactapi.ArtifactTypeModel),
		CreatedAt:     ts(1000),
		ModifiedAt:    ts(2000),
		DownloadCount: 42,
	}

	result := GetArtifactSummary(artifact)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Equal(t, "test-image", result.Data.ImageName)
	require.Equal(t, artifactapi.PackageTypeDOCKER, result.Data.PackageType)
	require.Equal(t, toPtr(artifactapi.ArtifactTypeModel), result.Data.ArtifactType)
	require.EqualValues(t, 42, *result.Data.DownloadsCount)
}

func TestGetArtifactVersionSummary_WithQuarantine(t *testing.T) {
	result := GetArtifactVersionSummary(
		"test-artifact",
		artifactapi.PackageTypeMAVEN,
		"2.0.0",
		true,
		"security vulnerability detected",
		toPtr(artifactapi.ArtifactTypeModel),
	)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Equal(t, "test-artifact", result.Data.ImageName)
	require.Equal(t, artifactapi.PackageTypeMAVEN, result.Data.PackageType)
	require.Equal(t, "2.0.0", result.Data.Version)
	require.EqualValues(t, true, *result.Data.IsQuarantined)
	require.Equal(t, "security vulnerability detected", *result.Data.QuarantineReason)
	require.Equal(t, toPtr(artifactapi.ArtifactTypeModel), result.Data.ArtifactType)
}

func TestGetAllArtifactResponse_EmptyList(t *testing.T) {
	ctx := context.Background()
	p := &fakeURLProvider{base: "https://empty.registry"}

	result := GetAllArtifactResponse(ctx, nil, 0, 1, 20, "root", p, "Bearer")

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Empty(t, result.Data.Artifacts)
	require.EqualValues(t, 0, *result.Data.ItemCount)
	require.EqualValues(t, 0, *result.Data.PageCount)
	require.EqualValues(t, 1, *result.Data.PageIndex)
	require.EqualValues(t, 20, *result.Data.PageSize)
}

func TestGetAllArtifactByRegistryResponse_WithPagination(t *testing.T) {
	artifacts := []types.ArtifactMetadata{
		{
			RepoName:      "repo1",
			Name:          "artifact1",
			LatestVersion: "1.0.0",
			ModifiedAt:    ts(1000),
			PackageType:   artifactapi.PackageTypeNPM,
			DownloadCount: 10,
		},
		{
			RepoName:      "repo2",
			Name:          "artifact2",
			LatestVersion: "2.0.0",
			ModifiedAt:    ts(2000),
			PackageType:   artifactapi.PackageTypePYTHON,
			DownloadCount: 20,
		},
	}

	result := GetAllArtifactByRegistryResponse(&artifacts, 2, 1, 10)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Len(t, result.Data.Artifacts, 2)
	require.EqualValues(t, 2, *result.Data.ItemCount)
	require.EqualValues(t, 1, *result.Data.PageCount)
	require.EqualValues(t, 1, *result.Data.PageIndex)
	require.EqualValues(t, 10, *result.Data.PageSize)
}

func TestGetAllArtifactLabelsResponse_WithLabels(t *testing.T) {
	labels := []string{"production", "staging", "development"}

	result := GetAllArtifactLabelsResponse(&labels, 3, 1, 20)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Equal(t, labels, result.Data.Labels)
	require.EqualValues(t, 3, *result.Data.ItemCount)
	require.EqualValues(t, 1, *result.Data.PageCount)
	require.EqualValues(t, 1, *result.Data.PageIndex)
	require.EqualValues(t, 20, *result.Data.PageSize)
}

func TestGetQuarantinePathJSONResponse_Complete(t *testing.T) {
	result := GetQuarantinePathJSONResponse(
		"quarantine-123",
		1,
		2,
		3,
		"malware detected",
		"/path/to/file",
	)

	require.Equal(t, artifactapi.StatusSUCCESS, result.Status)
	require.Equal(t, "quarantine-123", result.Data.Id)
	require.EqualValues(t, 1, result.Data.RegistryId)
	require.EqualValues(t, 2, result.Data.ArtifactId)
	require.EqualValues(t, 3, *result.Data.VersionId)
	require.Equal(t, "malware detected", result.Data.Reason)
	require.Equal(t, "/path/to/file", *result.Data.FilePath)
}
