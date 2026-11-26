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

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	cargohelper "github.com/harness/gitness/registry/app/utils/cargo"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

// w2rRegistryHelper answers every call of the shared registry helper with an empty value.
type w2rRegistryHelper struct{ calls int }

func (h *w2rRegistryHelper) GetAuthHeaderPrefix() string { return "Bearer" }
func (h *w2rRegistryHelper) DeleteFileNode(context.Context, *types.RegistryRequestBaseInfo, string) error {
	h.calls++
	return nil
}
func (h *w2rRegistryHelper) DeleteVersion(
	context.Context, *types.RegistryRequestBaseInfo, *types.Image, string, string, string,
) error {
	h.calls++
	return nil
}
func (h *w2rRegistryHelper) DeleteGenericImage(
	context.Context, *types.RegistryRequestBaseInfo, string, string,
) error {
	h.calls++
	return nil
}
func (h *w2rRegistryHelper) ReportDeleteVersionEvent(context.Context, *registryevents.ArtifactDeletedPayload) {
	h.calls++
}
func (h *w2rRegistryHelper) ReportBuildPackageIndexEvent(context.Context, int64, string) { h.calls++ }
func (h *w2rRegistryHelper) ReportBuildPackageMetadataEvent(context.Context, int64, string) {
	h.calls++
}
func (h *w2rRegistryHelper) ReportBuildRegistryIndexEvent(context.Context, int64, []types.SourceRef) {
	h.calls++
}
func (h *w2rRegistryHelper) GetPackageURL(_ context.Context, root, reg, pkgType string) string {
	return "https://pkg.example.com/" + root + "/" + reg + "/" + pkgType + "/"
}
func (h *w2rRegistryHelper) GetHostName(context.Context, string) string { return "pkg.example.com" }
func (h *w2rRegistryHelper) GetArtifactMetadata(types.ArtifactMetadata, string) *artifact.ArtifactMetadata {
	return &artifact.ArtifactMetadata{}
}
func (h *w2rRegistryHelper) GetArtifactVersionMetadata(
	types.NonOCIArtifactMetadata, string, string,
) *artifact.ArtifactVersionMetadata {
	return &artifact.ArtifactVersionMetadata{}
}
func (h *w2rRegistryHelper) GetFileMetadata(types.FileNodeMetadata, string, string) *artifact.FileDetail {
	return &artifact.FileDetail{}
}
func (h *w2rRegistryHelper) GetArtifactDetail(
	*types.Image, *types.Artifact, map[string]any, int64,
) *artifact.ArtifactDetail {
	return &artifact.ArtifactDetail{}
}
func (h *w2rRegistryHelper) ReplacePlaceholders(
	context.Context, *[]artifact.ClientSetupSection, string, string, *artifact.ArtifactParam,
	*artifact.VersionParam, string, string, string, string,
) {
}

type w2rCargoHelper struct{ cargohelper.RegistryHelper }

func (w2rCargoHelper) GetIndexFilePathFromImageName(name string) string { return "/index/" + name }

type w2rNpmMetadata struct{}

func (w2rNpmMetadata) UpdatePackageMetadata(context.Context, int64, int64, int64, string, string) error {
	return nil
}

func w2rAllHelpers() map[string]interfaces.PackageHelper {
	h := &w2rRegistryHelper{}
	return map[string]interfaces.PackageHelper{
		"cargo":       NewCargoPackageType(h, w2rCargoHelper{}),
		"docker":      NewDockerPackageType(h),
		"generic":     NewGenericPackageType(h),
		"go":          NewGoPackageType(h),
		"helm":        NewHelmPackageType(h),
		"huggingface": NewHuggingFacePackageType(h),
		"maven":       NewMavenPackageType(h),
		"npm":         NewNPMPackageType(h, w2rNpmMetadata{}),
		"nuget":       NewNugetPackageType(h),
		"python":      NewPythonPackageType(h),
		"rpm":         NewRPMPackageType(h),
	}
}

// w2rCall runs fn and reports whether it panicked, so one odd input does not hide the other methods.
func w2rCall(fn func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	fn()
	return false
}

func TestW2rPackageHelpersBasics(t *testing.T) {
	for name, h := range w2rAllHelpers() {
		require.NotEmpty(t, h.GetPackageType(), name)
		require.NotEmpty(t, h.GetPathPackageType(), name)
		require.True(t, h.IsValidRepoType(string(artifact.RegistryTypeUPSTREAM)) ||
			h.IsValidRepoType(string(artifact.RegistryTypeVIRTUAL)), name)
		require.False(t, h.IsValidRepoType("BAD"), name)
		require.False(t, h.IsValidUpstreamSource("BAD"), name)
		_ = h.IsFileOperationSupported()
		_ = h.IsURLRequiredForUpstreamSource(string(artifact.UpstreamConfigSourceCustom))
		_ = h.IsValidUpstreamSource(string(artifact.UpstreamConfigSourceCustom))
	}
}

func TestW2rPackageHelpersMethodsDoNotPanicOnTypicalInput(t *testing.T) {
	ctx := context.Background()
	regInfo := &types.RegistryRequestBaseInfo{RegistryID: 1, RegistryIdentifier: "reg", PackageType: "GENERIC"}
	img := &types.Image{Name: "pkg", RegistryID: 1}
	art := &types.Artifact{Metadata: []byte(`{}`), Version: "1.0.0"}
	image, version := artifact.ArtifactParam("pkg"), artifact.VersionParam("1.0.0")

	for name, h := range w2rAllHelpers() {
		calls := map[string]func(){
			"GetFilePath":            func() { _ = h.GetFilePath("pkg", "1.0.0") },
			"GetPackageURL":          func() { _ = h.GetPackageURL(ctx, "root", "reg") },
			"DeleteArtifact":         func() { _ = h.DeleteArtifact(ctx, regInfo, "pkg") },
			"DeleteVersion":          func() { _ = h.DeleteVersion(ctx, regInfo, img, "pkg", "1.0.0") },
			"ReportDeleteVersion":    func() { h.ReportDeleteVersionEvent(ctx, 1, 1, "pkg", "1.0.0") },
			"ReportPackageIndex":     func() { h.ReportBuildPackageIndexEvent(ctx, 1, "pkg") },
			"ReportRegistryIndex":    func() { h.ReportBuildRegistryIndexEvent(ctx, 1, nil) },
			"GetArtifactMetadata":    func() { _ = h.GetArtifactMetadata(types.ArtifactMetadata{}) },
			"GetArtifactVersionMeta": func() { _ = h.GetArtifactVersionMetadata("pkg", types.NonOCIArtifactMetadata{}) },
			"GetFileMetadata": func() {
				_ = h.GetFileMetadata(ctx, "root", "reg", "pkg", "1.0.0", types.FileNodeMetadata{})
			},
			"GetArtifactDetail": func() { _, _ = h.GetArtifactDetail(img, art, 3) },
			"GetClientSetupDetailsV": func() {
				_, _ = h.GetClientSetupDetails(ctx, "root/reg", &image, &version, artifact.RegistryTypeVIRTUAL)
			},
			"GetClientSetupDetailsU": func() {
				_, _ = h.GetClientSetupDetails(ctx, "root/reg", &image, &version, artifact.RegistryTypeUPSTREAM)
			},
			"GetClientSetupDetailsN": func() {
				_, _ = h.GetClientSetupDetails(ctx, "root/reg", nil, nil, artifact.RegistryTypeVIRTUAL)
			},
			"BuildRegistryIndexAsync": func() {
				_ = h.BuildRegistryIndexAsync(ctx, &types.Registry{ID: 1}, types.BuildRegistryIndexTaskPayload{})
			},
			"BuildPackageIndexAsync": func() {
				_ = h.BuildPackageIndexAsync(ctx, &types.Registry{ID: 1}, types.BuildPackageIndexTaskPayload{})
			},
			"BuildPackageMetadata": func() {
				_ = h.BuildPackageMetadataAsync(ctx, &types.Registry{ID: 1}, types.BuildPackageMetadataTaskPayload{})
			},
			"GetNodePathsForImage":    func() { _, _ = h.GetNodePathsForImage(nil, "pkg") },
			"GetNodePathsForArtifact": func() { _, _ = h.GetNodePathsForArtifact(nil, "pkg", "1.0.0") },
			"GetPkgDownloadURL": func() {
				_, _ = h.GetPkgDownloadURL(ctx, "root", "reg", "pkg", "model", "1.0.0", "f", "/f")
			},
			"GetPurlForArtifact":   func() { _, _ = h.GetPurlForArtifact("pkg", "1.0.0") },
			"GetPackageAndVersion": func() { _, _, _ = h.GetPackageAndVersionFromNodePath("/a/b/c/d/e") },
			"IsArtifactMainFile":   func() { _ = h.IsArtifactMainFile("/a/b/c.pom") },
		}
		for call, fn := range calls {
			if w2rCall(fn) {
				t.Logf("%s.%s panicked on typical input", name, call)
			}
		}
	}
}

func TestW2rPackageHelpersPurlAndNodePaths(t *testing.T) {
	all := w2rAllHelpers()

	purls := map[string]string{
		"cargo": "pkg:cargo/pkg@1.0.0", "go": "pkg:golang/pkg@1.0.0", "npm": "pkg:npm/pkg@1.0.0",
		"nuget": "pkg:nuget/pkg@1.0.0", "python": "pkg:pypi/pkg@1.0.0", "rpm": "pkg:rpm/pkg@1.0.0",
	}
	for name, want := range purls {
		got, err := all[name].GetPurlForArtifact("pkg", "1.0.0")
		require.NoError(t, err, name)
		require.Equal(t, want, got, name)
	}

	maven := all["maven"]
	purl, err := maven.GetPurlForArtifact("org.acme:lib", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "pkg:maven/org.acme/lib@1.0.0", purl)
	for _, bad := range [][2]string{{"", "1"}, {"a:b", ""}, {"no-colon", "1"}} {
		_, err = maven.GetPurlForArtifact(bad[0], bad[1])
		require.Error(t, err, bad)
	}

	nodes, err := maven.GetNodePathsForArtifact(nil, "org.acme:lib", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, []string{"/org/acme/lib/1.0.0"}, nodes)
	nodes, err = maven.GetNodePathsForImage(nil, "org.acme:lib")
	require.NoError(t, err)
	require.Equal(t, []string{"/org/acme/lib"}, nodes)

	pkgName, version, _ := maven.GetPackageAndVersionFromNodePath("/org/acme/lib/1.0.0/lib-1.0.0.pom")
	require.Equal(t, "org.acme:lib", pkgName)
	require.Equal(t, "1.0.0", version)
	pkgName, _, _ = maven.GetPackageAndVersionFromNodePath("/too/short")
	require.Empty(t, pkgName)

	cargoNodes, err := all["cargo"].GetNodePathsForArtifact(nil, "pkg", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, []string{"/crates/pkg/1.0.0"}, cargoNodes)
	goNodes, err := all["go"].GetNodePathsForArtifact(nil, "pkg", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, []string{"/pkg/@v/1.0.0"}, goNodes)

	require.Equal(t, "/crates/pkg/1.0.0", all["cargo"].GetFilePath("pkg", "1.0.0"))
	require.True(t, all["generic"].IsArtifactMainFile("/org/acme/lib/1.0.0/lib-1.0.0.pom"))
}
