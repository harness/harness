// Copyright 2026 Harness, Inc.
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

package nuget_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	urlprovider "github.com/harness/gitness/app/url"
	nugetctrl "github.com/harness/gitness/registry/app/api/controller/pkg/nuget"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	nugetpkg "github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rchRegistryDao struct {
	store.RegistryRepository
	reg *registrytypes.Registry
	err error
}

func (d *rchRegistryDao) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.reg, d.err
}

type rchQuarantine struct {
	quarantine.Finder
	err error
}

func (q rchQuarantine) CheckArtifactQuarantineStatus(
	_ context.Context, _ int64, _ string, _ string, _ *artifact.ArtifactType,
) error {
	return q.err
}

type rchURLProvider struct {
	urlprovider.Provider
}

func (rchURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return "http://pkg.example/" + regRef + "/" + pkgType
}

type rchNugetRegistry struct {
	nugetpkg.Registry
	err error
}

func (r *rchNugetRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchNugetRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNUGET}
}

func (r *rchNugetRegistry) UploadPackage(
	_ context.Context, _ nugettype.ArtifactInfo, _ io.ReadCloser, _ nugetpkg.FileBundleType,
) (*commons.ResponseHeaders, string, error) {
	return &commons.ResponseHeaders{Code: 201}, "", r.err
}

func (r *rchNugetRegistry) DownloadPackage(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, io.ReadCloser, error) {
	return &commons.ResponseHeaders{Code: 200}, nil, "http://r", io.NopCloser(strings.NewReader("x")), r.err
}

func (r *rchNugetRegistry) DeletePackage(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: 204}, r.err
}

func (r *rchNugetRegistry) ListPackageVersion(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*nugettype.PackageVersion, error) {
	return &nugettype.PackageVersion{Versions: []string{"1.0.0"}}, r.err
}

func (r *rchNugetRegistry) ListPackageVersionV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*nugettype.FeedResponse, error) {
	return &nugettype.FeedResponse{Count: 1}, r.err
}

func (r *rchNugetRegistry) CountPackageVersionV2(_ context.Context, _ nugettype.ArtifactInfo) (int64, error) {
	return 4, r.err
}

func (r *rchNugetRegistry) CountPackageV2(_ context.Context, _ nugettype.ArtifactInfo, _ string) (int64, error) {
	return 7, r.err
}

func (r *rchNugetRegistry) SearchPackageV2(
	_ context.Context, _ nugettype.ArtifactInfo, _ string, _ int, _ int,
) (*nugettype.FeedResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &nugettype.FeedResponse{
		Count:   1,
		Entries: []*nugettype.FeedEntryResponse{{ID: "e1"}},
	}, nil
}

func (r *rchNugetRegistry) SearchPackage(
	_ context.Context, _ nugettype.ArtifactInfo, _ string, _ int, _ int,
) (*nugettype.SearchResultResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &nugettype.SearchResultResponse{
		TotalHits: 1,
		Data:      []*nugettype.SearchResult{{ID: "pkg", Version: "1.0.0"}},
	}, nil
}

func (r *rchNugetRegistry) GetPackageMetadata(
	_ context.Context, _ nugettype.ArtifactInfo,
) (nugettype.RegistrationResponse, error) {
	return &nugettype.RegistrationIndexResponse{Count: 2}, r.err
}

func (r *rchNugetRegistry) GetPackageVersionMetadataV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*nugettype.FeedEntryResponse, error) {
	return &nugettype.FeedEntryResponse{ID: "entry"}, r.err
}

func (r *rchNugetRegistry) GetPackageVersionMetadata(
	_ context.Context, _ nugettype.ArtifactInfo,
) (*nugettype.RegistrationLeafResponse, error) {
	return &nugettype.RegistrationLeafResponse{Listed: true}, r.err
}

func (r *rchNugetRegistry) GetServiceEndpoint(_ context.Context, _ nugettype.ArtifactInfo) *nugettype.ServiceEndpoint {
	return &nugettype.ServiceEndpoint{Version: "3.0.0"}
}

func (r *rchNugetRegistry) GetServiceEndpointV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugettype.ServiceEndpointV2 {
	return &nugettype.ServiceEndpointV2{Base: "http://base"}
}

func (r *rchNugetRegistry) GetServiceMetadataV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugettype.ServiceMetadataV2 {
	return &nugettype.ServiceMetadataV2{Version: "1.0"}
}

func (r *rchNugetRegistry) GetReadme(_ context.Context, _ nugettype.ArtifactInfo) (string, error) {
	return "# readme", r.err
}

func rchNewInfo() nugettype.ArtifactInfo {
	return nugettype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5, RootIdentifier: "root"},
			RegIdentifier: "reg",
			Image:         "pkg",
		},
		Version:  "1.0.0",
		Filename: "pkg.1.0.0.nupkg",
	}
}

func rchSetup(t *testing.T, daoErr, regErr, quarantineErr error) nugetctrl.Controller {
	t.Helper()
	base.Register(&rchNugetRegistry{err: regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "NUGET:VIRTUAL") })
	dao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: 9, Name: "reg", ParentID: 5,
			PackageType: artifact.PackageTypeNUGET, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: daoErr,
	}
	return nugetctrl.NewController(
		nil, dao, nil, nil, nil, nil, rchURLProvider{}, nil, nil, rchQuarantine{err: quarantineErr}, nil,
	)
}

type rchOp struct {
	name string
	call func(c nugetctrl.Controller) error
}

func rchOps() []rchOp {
	ctx := context.Background()
	info := rchNewInfo
	return []rchOp{
		{"upload", func(c nugetctrl.Controller) error {
			return c.UploadPackage(ctx, info(), io.NopCloser(strings.NewReader("d")), nugetpkg.DependencyFile).GetError()
		}},
		{"upload symbols", func(c nugetctrl.Controller) error {
			return c.UploadPackage(ctx, info(), io.NopCloser(strings.NewReader("d")), nugetpkg.SymbolsFile).GetError()
		}},
		{"download", func(c nugetctrl.Controller) error { return c.DownloadPackage(ctx, info()).GetError() }},
		{"delete", func(c nugetctrl.Controller) error { return c.DeletePackage(ctx, info()).GetError() }},
		{"service endpoint", func(c nugetctrl.Controller) error { return c.GetServiceEndpoint(ctx, info()).GetError() }},
		{"service endpoint v2", func(c nugetctrl.Controller) error {
			return c.GetServiceEndpointV2(ctx, info()).GetError()
		}},
		{"service metadata v2", func(c nugetctrl.Controller) error {
			return c.GetServiceMetadataV2(ctx, info()).GetError()
		}},
		{"list versions", func(c nugetctrl.Controller) error { return c.ListPackageVersion(ctx, info()).GetError() }},
		{"list versions v2", func(c nugetctrl.Controller) error {
			return c.ListPackageVersionV2(ctx, info()).GetError()
		}},
		{"package metadata", func(c nugetctrl.Controller) error { return c.GetPackageMetadata(ctx, info()).GetError() }},
		{"version metadata", func(c nugetctrl.Controller) error {
			return c.GetPackageVersionMetadata(ctx, info()).GetError()
		}},
		{"version metadata v2", func(c nugetctrl.Controller) error {
			return c.GetPackageVersionMetadataV2(ctx, info()).GetError()
		}},
		{"version count v2", func(c nugetctrl.Controller) error {
			return c.CountPackageVersionV2(ctx, info()).GetError()
		}},
		{"count v2", func(c nugetctrl.Controller) error { return c.CountPackageV2(ctx, info(), "q").GetError() }},
		{"readme", func(c nugetctrl.Controller) error { return c.GetReadme(ctx, info()).GetError() }},
	}
}

func TestRchNugetControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.NoError(t, op.call(rchSetup(t, nil, nil, nil)))
		})
	}
}

func TestRchNugetControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.Error(t, op.call(rchSetup(t, errors.New("missing"), nil, nil)))
		})
	}
}

func TestRchNugetControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		switch op.name {
		case "service endpoint", "service endpoint v2", "service metadata v2":
			// These registry calls cannot fail.
			continue
		}
		t.Run(op.name, func(t *testing.T) {
			require.ErrorIs(t, op.call(rchSetup(t, nil, regErr, nil)), regErr)
		})
	}
}

func TestRchNugetControllerQuarantinedDownload(t *testing.T) {
	c := rchSetup(t, nil, nil, usererror.ErrQuarantinedArtifact)
	err := c.DownloadPackage(context.Background(), rchNewInfo()).GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchNugetControllerPayloads(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, nil, nil, nil)

	dl := c.DownloadPackage(ctx, rchNewInfo())
	require.Equal(t, "http://r", dl.RedirectURL)
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())

	require.Equal(t, []string{"1.0.0"}, c.ListPackageVersion(ctx, rchNewInfo()).PackageVersion.Versions)
	require.Equal(t, int64(1), c.ListPackageVersionV2(ctx, rchNewInfo()).FeedResponse.Count)
	require.Equal(t, int64(4), c.CountPackageVersionV2(ctx, rchNewInfo()).Count)
	require.Equal(t, int64(7), c.CountPackageV2(ctx, rchNewInfo(), "term").Count)
	require.Equal(t, "# readme", c.GetReadme(ctx, rchNewInfo()).ReadmeContent)
	require.Equal(t, "text/markdown; charset=utf-8",
		c.GetReadme(ctx, rchNewInfo()).ResponseHeaders.Headers["Content-Type"])
	require.Equal(t, "3.0.0", c.GetServiceEndpoint(ctx, rchNewInfo()).ServiceEndpoint.Version)
	require.Equal(t, "http://base", c.GetServiceEndpointV2(ctx, rchNewInfo()).ServiceEndpoint.Base)
	require.Equal(t, "1.0", c.GetServiceMetadataV2(ctx, rchNewInfo()).ServiceMetadata.Version)
	require.True(t, c.GetPackageVersionMetadata(ctx, rchNewInfo()).RegistrationLeafResponse.Listed)
	require.Equal(t, "entry", c.GetPackageVersionMetadataV2(ctx, rchNewInfo()).FeedEntryResponse.ID)
}

func TestRchNugetSearch(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, nil, nil, nil)

	v3 := c.SearchPackage(ctx, rchNewInfo(), "pkg", 10, 0)
	require.NoError(t, v3.GetError())
	require.Equal(t, int64(1), v3.SearchResponse.TotalHits)
	require.Len(t, v3.SearchResponse.Data, 1)
	require.Equal(t, "pkg", v3.SearchResponse.Data[0].ID)

	// The page is full, so a "next" link must be generated.
	v2 := c.SearchPackageV2(ctx, rchNewInfo(), "pkg", 1, 0)
	require.NoError(t, v2.GetError())
	require.Len(t, v2.FeedResponse.Entries, 1)
	require.Len(t, v2.FeedResponse.Links, 2)
	require.Equal(t, "next", v2.FeedResponse.Links[1].Rel)
	require.Contains(t, string(v2.FeedResponse.Links[1].Href), "searchTerm=pkg")

	// The page is not full, so only the "self" link is returned.
	v2 = c.SearchPackageV2(ctx, rchNewInfo(), "", 10, 0)
	require.NoError(t, v2.GetError())
	require.Len(t, v2.FeedResponse.Links, 1)
	require.Equal(t, "self", v2.FeedResponse.Links[0].Rel)
}

func TestRchNugetSearchErrors(t *testing.T) {
	ctx := context.Background()

	missing := rchSetup(t, errors.New("missing"), nil, nil)
	require.Error(t, missing.SearchPackage(ctx, rchNewInfo(), "pkg", 10, 0).GetError())
	require.Error(t, missing.SearchPackageV2(ctx, rchNewInfo(), "pkg", 10, 0).GetError())

	// A failing registry is skipped by the aggregating search, leaving an empty result.
	failing := rchSetup(t, nil, errors.New("backend failed"), nil)
	v3 := failing.SearchPackage(ctx, rchNewInfo(), "pkg", 10, 0)
	require.NoError(t, v3.GetError())
	require.Empty(t, v3.SearchResponse.Data)
	v2 := failing.SearchPackageV2(ctx, rchNewInfo(), "pkg", 10, 0)
	require.NoError(t, v2.GetError())
	require.Empty(t, v2.FeedResponse.Entries)
}
