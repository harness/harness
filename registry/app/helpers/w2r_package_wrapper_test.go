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

package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/interfaces"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errW2rBoom = errors.New("boom")

// w2rPackage is a package helper that only implements what the wrapper calls.
type w2rPackage struct {
	interfaces.PackageHelper
	pathType    string
	repoTypes   map[string]bool
	sources     map[string]bool
	urlRequired bool
	fileOps     bool
	err         error
	events      []string
}

func (p *w2rPackage) GetPathPackageType() string     { return p.pathType }
func (p *w2rPackage) IsFileOperationSupported() bool { return p.fileOps }
func (p *w2rPackage) IsValidRepoType(r string) bool  { return p.repoTypes[r] }
func (p *w2rPackage) IsValidUpstreamSource(s string) bool {
	return p.sources[s]
}
func (p *w2rPackage) IsURLRequiredForUpstreamSource(string) bool { return p.urlRequired }
func (p *w2rPackage) GetFilePath(a, v string) string             { return "/" + a + "/" + v }
func (p *w2rPackage) GetPackageURL(_ context.Context, root, reg string) string {
	return "https://" + root + "/" + reg
}
func (p *w2rPackage) DeleteArtifact(context.Context, *types.RegistryRequestBaseInfo, string) error {
	return p.err
}
func (p *w2rPackage) DeleteVersion(
	context.Context, *types.RegistryRequestBaseInfo, *types.Image, string, string,
) error {
	return p.err
}
func (p *w2rPackage) ReportDeleteVersionEvent(_ context.Context, principalID, registryID int64, a, v string) {
	p.events = append(p.events, "delete-version")
	_ = principalID + registryID
	_, _ = a, v
}
func (p *w2rPackage) ReportBuildPackageIndexEvent(context.Context, int64, string) {
	p.events = append(p.events, "package-index")
}
func (p *w2rPackage) ReportBuildRegistryIndexEvent(context.Context, int64, []types.SourceRef) {
	p.events = append(p.events, "registry-index")
}
func (p *w2rPackage) GetArtifactMetadata(types.ArtifactMetadata) *artifactapi.ArtifactMetadata {
	return &artifactapi.ArtifactMetadata{}
}
func (p *w2rPackage) GetArtifactVersionMetadata(
	string, types.NonOCIArtifactMetadata,
) *artifactapi.ArtifactVersionMetadata {
	return &artifactapi.ArtifactVersionMetadata{}
}
func (p *w2rPackage) GetFileMetadata(
	context.Context, string, string, string, string, types.FileNodeMetadata,
) *artifactapi.FileDetail {
	return &artifactapi.FileDetail{}
}
func (p *w2rPackage) GetArtifactDetail(*types.Image, *types.Artifact, int64) (*artifactapi.ArtifactDetail, error) {
	return &artifactapi.ArtifactDetail{}, p.err
}
func (p *w2rPackage) GetClientSetupDetails(
	context.Context, string, *artifactapi.ArtifactParam, *artifactapi.VersionParam, artifactapi.RegistryType,
) (*artifactapi.ClientSetupDetails, error) {
	return &artifactapi.ClientSetupDetails{}, p.err
}
func (p *w2rPackage) BuildRegistryIndexAsync(
	context.Context, *types.Registry, types.BuildRegistryIndexTaskPayload,
) error {
	return p.err
}
func (p *w2rPackage) BuildPackageIndexAsync(
	context.Context, *types.Registry, types.BuildPackageIndexTaskPayload,
) error {
	return p.err
}
func (p *w2rPackage) BuildPackageMetadataAsync(
	context.Context, *types.Registry, types.BuildPackageMetadataTaskPayload,
) error {
	return p.err
}
func (p *w2rPackage) GetNodePathsForImage(*string, string) ([]string, error) {
	return []string{"a"}, p.err
}
func (p *w2rPackage) GetNodePathsForArtifact(*string, string, string) ([]string, error) {
	return []string{"a", "b"}, p.err
}
func (p *w2rPackage) GetPkgDownloadURL(
	context.Context, string, string, string, string, string, string, string,
) (string, error) {
	return "https://dl", p.err
}
func (p *w2rPackage) GetPurlForArtifact(n, v string) (string, error) {
	return "pkg:" + n + "@" + v, p.err
}
func (p *w2rPackage) GetPackageAndVersionFromNodePath(string) (string, string, string) {
	return "pkg", "1.0", "type"
}
func (p *w2rPackage) IsArtifactMainFile(string) bool { return true }

type w2rFactory struct{ pkgs map[string]*w2rPackage }

func (f *w2rFactory) Register(interfaces.PackageHelper) {}
func (f *w2rFactory) Get(t string) interfaces.PackageHelper {
	if p, ok := f.pkgs[t]; ok {
		return p
	}
	return nil
}
func (f *w2rFactory) GetAllPackageTypes() []string {
	out := make([]string, 0, len(f.pkgs))
	for t := range f.pkgs {
		out = append(out, t)
	}
	return out
}
func (f *w2rFactory) IsValidPackageType(t string) bool { _, ok := f.pkgs[t]; return ok }

type w2rRegFinder struct {
	refcache.RegistryFinder
	reg *types.Registry
	err error
}

func (f w2rRegFinder) FindByID(context.Context, int64) (*types.Registry, error) { return f.reg, f.err }

func w2rWrapper(pkg *w2rPackage, finder w2rRegFinder) interfaces.PackageWrapper {
	return NewPackageWrapper(&w2rFactory{pkgs: map[string]*w2rPackage{"GENERIC": pkg}}, finder)
}

func TestW2rPackageWrapperValidation(t *testing.T) {
	pkg := &w2rPackage{
		pathType: "generic", repoTypes: map[string]bool{"VIRTUAL": true}, sources: map[string]bool{"Custom": true},
		urlRequired: true, fileOps: true,
	}
	w := w2rWrapper(pkg, w2rRegFinder{})

	require.True(t, w.IsFileOperationSupported("GENERIC"))
	require.False(t, w.IsFileOperationSupported(""))
	require.False(t, w.IsFileOperationSupported("MISSING"))

	require.True(t, w.IsValidPackageType(""))
	require.True(t, w.IsValidPackageType("GENERIC"))
	require.False(t, w.IsValidPackageType("MISSING"))
	require.True(t, w.IsValidPackageTypes([]string{"GENERIC", ""}))
	require.False(t, w.IsValidPackageTypes([]string{"GENERIC", "MISSING"}))

	require.True(t, w.IsValidRepoType(""))
	require.True(t, w.IsValidRepoType("VIRTUAL"))
	require.False(t, w.IsValidRepoType("UPSTREAM"))
	require.True(t, w.IsValidRepoTypes([]string{"VIRTUAL"}))
	require.False(t, w.IsValidRepoTypes([]string{"VIRTUAL", "UPSTREAM"}))
	require.True(t, w.ValidateRepoType("GENERIC", "VIRTUAL"))
	require.False(t, w.ValidateRepoType("MISSING", "VIRTUAL"))

	require.True(t, w.IsValidUpstreamSource(""))
	require.True(t, w.IsValidUpstreamSource("Custom"))
	require.False(t, w.IsValidUpstreamSource("Other"))
	require.True(t, w.IsValidUpstreamSources([]string{"Custom"}))
	require.False(t, w.IsValidUpstreamSources([]string{"Custom", "Other"}))
	require.True(t, w.ValidateUpstreamSource("GENERIC", "Custom"))
	require.False(t, w.ValidateUpstreamSource("MISSING", "Custom"))
	require.True(t, w.IsURLRequiredForUpstreamSource("GENERIC", "Custom"))
	require.False(t, w.IsURLRequiredForUpstreamSource("MISSING", "Custom"))

	got, err := w.GetPackageTypeFromPathPackageType("generic")
	require.NoError(t, err)
	require.Equal(t, "GENERIC", got)
	_, err = w.GetPackageTypeFromPathPackageType("unknown")
	require.Error(t, err)
}

func TestW2rPackageWrapperDelegation(t *testing.T) {
	ctx := context.Background()
	pkg := &w2rPackage{}
	reg := &types.Registry{ID: 7, PackageType: artifactapi.PackageType("GENERIC")}
	w := w2rWrapper(pkg, w2rRegFinder{reg: reg})
	info := &types.RegistryRequestBaseInfo{PackageType: "GENERIC", RegistryID: 7}

	require.NoError(t, w.DeleteArtifactVersion(ctx, info, &types.Image{}, "a", "1"))
	require.Error(t, w2rWrapper(&w2rPackage{err: errW2rBoom}, w2rRegFinder{}).
		DeleteArtifactVersion(ctx, info, &types.Image{}, "a", "1"))
	require.Error(t, w.DeleteArtifactVersion(ctx, &types.RegistryRequestBaseInfo{PackageType: "X"}, nil, "a", "1"))

	path, err := w.GetFilePath("GENERIC", "a", "1")
	require.NoError(t, err)
	require.Equal(t, "/a/1", path)
	_, err = w.GetFilePath("X", "a", "1")
	require.Error(t, err)

	url, err := w.GetPackageURL(ctx, "root", "reg", "GENERIC")
	require.NoError(t, err)
	require.Equal(t, "https://root/reg", url)
	_, err = w.GetPackageURL(ctx, "root", "reg", "X")
	require.Error(t, err)

	require.NotNil(t, w.GetArtifactMetadata(types.ArtifactMetadata{PackageType: "GENERIC"}))
	require.Nil(t, w.GetArtifactMetadata(types.ArtifactMetadata{PackageType: "X"}))
	require.NotNil(t, w.GetArtifactVersionMetadata("GENERIC", "img", types.NonOCIArtifactMetadata{}))
	require.Nil(t, w.GetArtifactVersionMetadata("X", "img", types.NonOCIArtifactMetadata{}))
	require.NotNil(t, w.GetFileMetadata(ctx, "r", "g", "GENERIC", "a", "1", types.FileNodeMetadata{}))
	require.Nil(t, w.GetFileMetadata(ctx, "r", "g", "X", "a", "1", types.FileNodeMetadata{}))

	detail, err := w.GetArtifactDetail("GENERIC", &types.Image{}, &types.Artifact{}, 1)
	require.NoError(t, err)
	require.NotNil(t, detail)
	_, err = w.GetArtifactDetail("X", nil, nil, 0)
	require.Error(t, err)

	setup, err := w.GetClientSetupDetails(ctx, "ref", nil, nil, artifactapi.RegistryType("VIRTUAL"), "GENERIC")
	require.NoError(t, err)
	require.NotNil(t, setup)
	_, err = w.GetClientSetupDetails(ctx, "ref", nil, nil, artifactapi.RegistryType("VIRTUAL"), "X")
	require.Error(t, err)

	nodes, err := w.GetNodePathsForImage("GENERIC", nil, "p")
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, nodes)
	nodes, err = w.GetNodePathsForImage("X", nil, "p")
	require.NoError(t, err)
	require.Empty(t, nodes)
	nodes, err = w.GetNodePathsForArtifact("GENERIC", nil, "p", "1")
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	nodes, err = w.GetNodePathsForArtifact("X", nil, "p", "1")
	require.NoError(t, err)
	require.Empty(t, nodes)

	dl, err := w.GetPkgDownloadURL(ctx, "GENERIC", "r", "g", "p", "t", "1", "f", "fp")
	require.NoError(t, err)
	require.Equal(t, "https://dl", dl)
	_, err = w.GetPkgDownloadURL(ctx, "X", "r", "g", "p", "t", "1", "f", "fp")
	require.Error(t, err)

	purl, err := w.GetPurlForArtifact("GENERIC", "p", "1")
	require.NoError(t, err)
	require.Equal(t, "pkg:p@1", purl)
	_, err = w.GetPurlForArtifact("X", "p", "1")
	require.Error(t, err)

	name, version, artType, err := w.GetPackageAndVersionFromNodePath("GENERIC", "/x")
	require.NoError(t, err)
	require.Equal(t, []string{"pkg", "1.0", "type"}, []string{name, version, artType})
	_, _, _, err = w.GetPackageAndVersionFromNodePath("X", "/x")
	require.Error(t, err)
}

func TestW2rPackageWrapperEventsAndAsyncBuilds(t *testing.T) {
	pkg := &w2rPackage{}
	reg := &types.Registry{ID: 7, PackageType: artifactapi.PackageType("GENERIC")}
	w := w2rWrapper(pkg, w2rRegFinder{reg: reg})
	session := &auth.Session{Principal: gitnesstypes.Principal{ID: 3}}
	ctx := request.WithAuthSession(context.Background(), session)

	require.NoError(t, w.TriggerIndexEvents(ctx, 7, "a", "1"))
	require.Equal(t, []string{"delete-version", "package-index", "registry-index"}, pkg.events)

	// Reporting a delete needs an authenticated session.
	require.Error(t, w.ReportDeleteVersionEvent(context.Background(), 7, "a", "1"))
	require.Error(t, w.TriggerIndexEvents(context.Background(), 7, "a", "1"))

	require.NoError(t, w.DeleteArtifact(ctx, &types.RegistryRequestBaseInfo{PackageType: "GENERIC", RegistryID: 7}, "a"))
	require.Error(t, w.DeleteArtifact(ctx, &types.RegistryRequestBaseInfo{PackageType: "X"}, "a"))
	require.Error(t, w2rWrapper(&w2rPackage{err: errW2rBoom}, w2rRegFinder{reg: reg}).
		DeleteArtifact(ctx, &types.RegistryRequestBaseInfo{PackageType: "GENERIC", RegistryID: 7}, "a"))

	require.NoError(t, w.BuildRegistryIndexAsync(ctx, types.BuildRegistryIndexTaskPayload{RegistryID: 7}))
	require.NoError(t, w.BuildPackageIndexAsync(ctx, types.BuildPackageIndexTaskPayload{RegistryID: 7}))
	require.NoError(t, w.BuildPackageMetadataAsync(ctx, types.BuildPackageMetadataTaskPayload{RegistryID: 7}))
	require.NoError(t, w.ReportBuildPackageIndexEvent(ctx, 7, "a"))
	require.NoError(t, w.ReportBuildRegistryIndexEvent(ctx, 7, nil))
	require.True(t, w.IsArtifactMainFile("GENERIC", "/x"))
}

func TestW2rPackageWrapperRegistryLookupFailures(t *testing.T) {
	ctx := request.WithAuthSession(context.Background(), &auth.Session{})
	failing := w2rWrapper(&w2rPackage{}, w2rRegFinder{err: errW2rBoom})
	unknownType := w2rWrapper(&w2rPackage{}, w2rRegFinder{reg: &types.Registry{PackageType: "X"}})

	for name, w := range map[string]interfaces.PackageWrapper{"lookup fails": failing, "unknown type": unknownType} {
		require.Error(t, w.ReportDeleteVersionEvent(ctx, 1, "a", "1"), name)
		require.Error(t, w.ReportBuildPackageIndexEvent(ctx, 1, "a"), name)
		require.Error(t, w.ReportBuildRegistryIndexEvent(ctx, 1, nil), name)
		require.Error(t, w.BuildRegistryIndexAsync(ctx, types.BuildRegistryIndexTaskPayload{}), name)
		require.Error(t, w.BuildPackageIndexAsync(ctx, types.BuildPackageIndexTaskPayload{}), name)
		require.Error(t, w.BuildPackageMetadataAsync(ctx, types.BuildPackageMetadataTaskPayload{}), name)
	}
}
