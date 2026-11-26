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

package nuget

import (
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"

	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

const rpfBaseURL = "https://example.test/pkg/nuget"

func rpfMetadataJSON(t *testing.T, m nugetmetadata.NugetMetadata) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return raw
}

func rpfSampleMetadata() nugetmetadata.NugetMetadata {
	return nugetmetadata.NugetMetadata{
		Metadata: nugetmetadata.Metadata{
			PackageMetadata: nugetmetadata.PackageMetadata{
				ID:          "demo",
				Version:     "1.0.0",
				Title:       "Demo Title",
				Authors:     "alice",
				Description: "a demo package",
				ProjectURL:  "https://example.test/project",
				LicenseURL:  "https://example.test/license",
				Tags:        "one two",
				License:     &nugetmetadata.License{Type: "expression", Text: "MIT"},
				Dependencies: &nugetmetadata.DependenciesWrapper{
					Groups: []nugetmetadata.DependencyGroup{
						{
							TargetFramework: "net6.0",
							Dependencies: []nugetmetadata.Dependency{
								{ID: "dep.a", Version: "1.0"},
								{ID: "", Version: "2.0"},
							},
						},
						{TargetFramework: "net8.0"},
					},
					Dependencies: []nugetmetadata.Dependency{
						{ID: "dep.b", Version: "3.0"},
						{ID: "dep.c"},
					},
				},
			},
		},
		Size: 42,
	}
}

func TestRpfValidateAndNormaliseVersion(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "1", want: "1.0.0"},
		{in: "1.2", want: "1.2.0"},
		{in: "1.2.3", want: "1.2.3"},
		{in: "1.02.003", want: "1.2.3"},
		{in: "1.2.3.0", want: "1.2.3"},
		{in: "1.2.3.4", want: "1.2.3.4"},
		{in: "v2.0.1", want: "2.0.1"},
		{in: "1.0.0-beta.1", want: "1.0.0-beta.1"},
		{in: "1.0-rc1+build5", want: "1.0.0-rc1"},
		{in: "1.2.3.4.5", wantErr: true},
		{in: "abc", wantErr: true},
		{in: "", wantErr: true},
		{in: "99999999999999999999.1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := validateAndNormaliseVersion(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRpfURLBuilders(t *testing.T) {
	require.Equal(t, rpfBaseURL+"/registration/demo/index.json", getRegistrationIndexURL(rpfBaseURL, "demo"))
	require.Equal(t, rpfBaseURL+"/registration/demo/1.0.0.json", getRegistrationLeafURL(rpfBaseURL, "demo", "1.0.0"))
	require.Equal(t, rpfBaseURL+"/package/demo/1.0.0/demo.1.0.0.nupkg",
		getPackageDownloadURL(rpfBaseURL, "demo", "1.0.0"))
	require.Equal(t, rpfBaseURL+"/demo/1.0.0/readme", getReadmeURL(rpfBaseURL, "demo", "1.0.0"))
	require.Equal(t, rpfBaseURL+"/Packages(Id='demo',Version='1.0.0')",
		getPackageMetadataURL(rpfBaseURL, "demo", "1.0.0"))
	require.Equal(t, "https://a/b?proxy_endpoint=https://up/x", getProxyURL("https://a/b", "https://up/x"))

	inner := getInnerXMLField(rpfBaseURL, "demo", "1.0.0")
	require.Contains(t, inner, "<id>"+rpfBaseURL+"/Packages(Id='demo',Version='1.0.0')</id>")
	require.Contains(t, inner, `src="`+rpfBaseURL+`/package/demo/1.0.0/demo.1.0.0.nupkg"`)
}

func TestRpfBuildServiceEndpoints(t *testing.T) {
	v3 := buildServiceEndpoint(rpfBaseURL)
	require.Equal(t, "3.0.0", v3.Version)
	require.Len(t, v3.Resources, 10)
	types3 := map[string]string{}
	for _, r := range v3.Resources {
		types3[r.Type] = r.ID
	}
	require.Equal(t, rpfBaseURL+"/query", types3["SearchQueryService"])
	require.Equal(t, rpfBaseURL+"/package", types3["PackageBaseAddress/3.0.0"])
	require.Equal(t, rpfBaseURL, types3["PackagePublish/2.0.0"])

	v2 := buildServiceV2Endpoint(rpfBaseURL)
	require.Equal(t, rpfBaseURL, v2.Base)
	require.Equal(t, XMLNamespaceApp, v2.Xmlns)
	require.Equal(t, "Packages", v2.Workspace.Collection.Href)
	out, err := xml.Marshal(v2)
	require.NoError(t, err)
	require.Contains(t, string(out), "<service")
}

func TestRpfGetServiceMetadataV2(t *testing.T) {
	md := getServiceMetadataV2()
	require.NotNil(t, md)
	require.NotEmpty(t, md.DataServices.Schema)
	out, err := xml.Marshal(md)
	require.NoError(t, err)
	require.Contains(t, string(out), "V2FeedPackage")
	require.Contains(t, string(out), "FindPackagesById")
}

func TestRpfBuildDependencyString(t *testing.T) {
	m := rpfSampleMetadata()
	require.Equal(t, "dep.a:1.0:net6.0|:2.0:net6.0", buildDependencyString(&m))

	empty := nugetmetadata.NugetMetadata{}
	require.Empty(t, buildDependencyString(&empty))

	noGroups := nugetmetadata.NugetMetadata{Metadata: nugetmetadata.Metadata{
		PackageMetadata: nugetmetadata.PackageMetadata{Dependencies: &nugetmetadata.DependenciesWrapper{}},
	}}
	require.Empty(t, buildDependencyString(&noGroups))
}

func TestRpfCreateDependencyGroups(t *testing.T) {
	empty := nugetmetadata.NugetMetadata{}
	require.Nil(t, createDependencyGroups(&empty))

	m := rpfSampleMetadata()
	groups := createDependencyGroups(&m)
	require.Len(t, groups, 3)
	require.Equal(t, "net6.0", groups[0].TargetFramework)
	require.Equal(t, []*nugettype.PackageDependency{{ID: "dep.a", Range: "1.0"}}, groups[0].Dependencies)
	require.Equal(t, "net8.0", groups[1].TargetFramework)
	require.Nil(t, groups[1].Dependencies)
	require.Empty(t, groups[2].TargetFramework)
	require.Equal(t, []*nugettype.PackageDependency{{ID: "dep.b", Range: "3.0"}}, groups[2].Dependencies)

	onlyInvalid := nugetmetadata.NugetMetadata{Metadata: nugetmetadata.Metadata{
		PackageMetadata: nugetmetadata.PackageMetadata{Dependencies: &nugetmetadata.DependenciesWrapper{
			Dependencies: []nugetmetadata.Dependency{{ID: "x"}},
		}},
	}}
	require.Empty(t, createDependencyGroups(&onlyInvalid))
}

func TestRpfCreateRegistrationIndexResponse(t *testing.T) {
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	meta := rpfMetadataJSON(t, rpfSampleMetadata())
	artifacts := []types.Artifact{
		{Version: "2.0.0", Metadata: meta, CreatedAt: created},
		{Version: "1.0.0", Metadata: meta, CreatedAt: created},
	}
	info := nugettype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}

	resp, err := createRegistrationIndexResponse(rpfBaseURL, info, &artifacts)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	require.Equal(t, getRegistrationIndexURL(rpfBaseURL, "demo"), resp.RegistrationIndexURL)
	require.Len(t, resp.Pages, 1)
	page := resp.Pages[0]
	require.Equal(t, 2, page.Count)
	require.Equal(t, "1.0.0", page.Lower)
	require.Equal(t, "2.0.0", page.Upper)
	require.Len(t, page.Items, 2)
	item := page.Items[0]
	require.Equal(t, getRegistrationLeafURL(rpfBaseURL, "demo", "1.0.0"), item.RegistrationLeafURL)
	require.Equal(t, "MIT", item.CatalogEntry.LicenseExpression)
	require.Equal(t, "2024-01-02T03:04:05Z", item.CatalogEntry.Published)
	require.Equal(t, getReadmeURL(rpfBaseURL, "demo", "1.0.0"), item.CatalogEntry.ReadmeURL)
	require.Len(t, item.CatalogEntry.DependencyGroups, 3)

	bad := []types.Artifact{{Version: "1.0.0", Metadata: json.RawMessage("{")}}
	_, err = createRegistrationIndexResponse(rpfBaseURL, info, &bad)
	require.Error(t, err)
}

func TestRpfCreateRegistrationIndexPageItemNoLicense(t *testing.T) {
	m := rpfSampleMetadata()
	m.PackageMetadata.License = nil
	artifact := &types.Artifact{Version: "1.0.0", Metadata: rpfMetadataJSON(t, m)}
	info := nugettype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}
	item, err := createRegistrationIndexPageItem(rpfBaseURL, info, artifact)
	require.NoError(t, err)
	require.Empty(t, item.CatalogEntry.LicenseExpression)
	require.Equal(t, "demo", item.CatalogEntry.ID)
}

func TestRpfCreateRegistrationLeafResponse(t *testing.T) {
	created := time.Date(2024, 5, 6, 0, 0, 0, 0, time.UTC)
	info := nugettype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}, Version: "1.0.0"}
	resp := createRegistrationLeafResponse(rpfBaseURL, info, &types.Artifact{CreatedAt: created})
	require.True(t, resp.Listed)
	require.Equal(t, created, resp.Published)
	require.Equal(t, getRegistrationLeafURL(rpfBaseURL, "demo", "1.0.0"), resp.RegistrationLeafURL)
	require.Equal(t, getPackageDownloadURL(rpfBaseURL, "demo", "1.0.0"), resp.PackageContentURL)
	require.Equal(t, getRegistrationIndexURL(rpfBaseURL, "demo"), resp.RegistrationIndexURL)
	require.Equal(t, []string{"Package", "http://schema.nuget.org/catalog#Permalink"}, resp.Type)
}

func TestRpfCreateFeedEntryAndFeedResponse(t *testing.T) {
	m := rpfSampleMetadata()
	meta := rpfMetadataJSON(t, m)
	info := nugettype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}

	entry, err := createFeedEntryResponse(rpfBaseURL, info, &types.Artifact{Version: "1.0.0", Metadata: meta})
	require.NoError(t, err)
	require.Equal(t, "Demo Title", entry.Title.Value)
	require.Equal(t, "alice", entry.Author)
	require.Equal(t, int64(42), entry.Properties.PackageSize.Value)
	require.Equal(t, "demo", entry.Properties.ID)
	require.Contains(t, entry.Properties.Dependencies, "dep.a:1.0:net6.0")
	require.Contains(t, entry.Content, "demo.1.0.0.nupkg")

	// A missing title falls back to the image name.
	m.PackageMetadata.Title = ""
	entry, err = createFeedEntryResponse(rpfBaseURL, info,
		&types.Artifact{Version: "1.0.0", Metadata: rpfMetadataJSON(t, m)})
	require.NoError(t, err)
	require.Equal(t, "demo", entry.Title.Value)

	_, err = createFeedEntryResponse(rpfBaseURL, info, &types.Artifact{Metadata: json.RawMessage("nope")})
	require.Error(t, err)

	artifacts := []types.Artifact{
		{Version: "2.0.0", Metadata: meta},
		{Version: "1.0.0", Metadata: meta},
	}
	feed, err := createFeedResponse(rpfBaseURL, info, &artifacts)
	require.NoError(t, err)
	require.Equal(t, int64(2), feed.Count)
	require.Len(t, feed.Entries, 2)
	require.Equal(t, "1.0.0", feed.Entries[0].Properties.Version)
	require.Len(t, feed.Links, 1)
	require.Equal(t, "self", feed.Links[0].Rel)

	badArtifacts := []types.Artifact{{Version: "1", Metadata: json.RawMessage("x")}}
	_, err = createFeedResponse(rpfBaseURL, info, &badArtifacts)
	require.Error(t, err)
}

func TestRpfCreateSearchV2Response(t *testing.T) {
	meta := rpfMetadataJSON(t, rpfSampleMetadata())

	empty, err := createSearchV2Response(rpfBaseURL, nil, "term", 10, 0)
	require.NoError(t, err)
	require.Zero(t, empty.Count)
	require.Empty(t, empty.Entries)

	artifacts := []types.ArtifactMetadata{
		{Name: "demo", Version: "1.0.0", Metadata: meta},
		{Name: "other", Version: "2.0.0", Metadata: meta},
	}

	// Fewer results than the limit means there is no next page.
	resp, err := createSearchV2Response(rpfBaseURL, &artifacts, "term", 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.Count)
	require.Len(t, resp.Links, 1)
	require.Len(t, resp.Entries, 2)

	// A full page adds a next link with the paging parameters.
	resp, err = createSearchV2Response(rpfBaseURL, &artifacts, "term", 2, 4)
	require.NoError(t, err)
	require.Len(t, resp.Links, 2)
	require.Equal(t, "next", resp.Links[1].Rel)
	next := string(resp.Links[1].Href)
	require.Contains(t, next, "Search()")
	require.Contains(t, next, "%24skip=6")
	require.Contains(t, next, "%24top=2")
	require.Contains(t, next, "searchTerm=term")

	resp, err = createSearchV2Response(rpfBaseURL, &artifacts, "", 2, 0)
	require.NoError(t, err)
	require.NotContains(t, string(resp.Links[1].Href), "searchTerm")

	bad := []types.ArtifactMetadata{{Name: "demo", Metadata: json.RawMessage("x")}}
	_, err = createSearchV2Response(rpfBaseURL, &bad, "", 10, 0)
	require.Error(t, err)
}

func TestRpfCreateSearchResponse(t *testing.T) {
	meta := rpfMetadataJSON(t, rpfSampleMetadata())

	resp, err := createSearchResponse(rpfBaseURL, nil, 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), resp.TotalHits)
	require.Empty(t, resp.Data)

	artifacts := []types.ArtifactMetadata{
		{Name: "a", Version: "1.0.0", Metadata: meta},
		{Name: "a", Version: "2.0.0", Metadata: meta},
		{Name: "b", Version: "1.5.0", Metadata: meta},
	}
	resp, err = createSearchResponse(rpfBaseURL, &artifacts, 2)
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.TotalHits)
	require.Len(t, resp.Data, 2)
	require.Equal(t, "a", resp.Data[0].ID)
	require.Equal(t, "2.0.0", resp.Data[0].Version)
	require.Len(t, resp.Data[0].Versions, 2)
	require.Equal(t, []string{"alice"}, resp.Data[0].Authors)
	require.Equal(t, "b", resp.Data[1].ID)
	require.Equal(t, "1.5.0", resp.Data[1].Version)
	require.Len(t, resp.Data[1].Versions, 1)
	require.Equal(t, getRegistrationIndexURL(rpfBaseURL, "b"), resp.Data[1].RegistrationIndexURL)

	bad := []types.ArtifactMetadata{
		{Name: "a", Version: "1", Metadata: json.RawMessage("x")},
		{Name: "b", Version: "1", Metadata: meta},
	}
	_, err = createSearchResponse(rpfBaseURL, &bad, 1)
	require.Error(t, err)
}

func TestRpfReplaceBaseWithURL(t *testing.T) {
	got, err := replaceBaseWithURL("https://www.nuget.org/api/v2/Packages(Id='a',Version='1')", "https://me/x")
	require.NoError(t, err)
	require.Equal(t, "https://me/x/Packages(Id='a',Version='1')", got)

	_, err = replaceBaseWithURL("https://www.nuget.org/api/v2/other", "https://me/x")
	require.Error(t, err)

	_, err = replaceBaseWithURL("not a url/Packages", "https://me/x")
	require.Error(t, err)
}

func TestRpfModifyContent(t *testing.T) {
	id := "https://www.nuget.org/api/v2/Packages(Id='a',Version='1.0.0')"
	src := "https://www.nuget.org/api/v2/package/a/1.0.0"
	feed := &nugettype.FeedEntryResponse{
		ID:              id,
		Content:         `<id>` + id + `</id><content src="` + src + `"/>`,
		DownloadContent: &nugettype.FeedEntryContent{Source: src},
	}
	require.NoError(t, modifyContent(feed, "https://me/x", "a", "1.0.0"))
	require.Empty(t, feed.ID)
	require.Nil(t, feed.DownloadContent)
	require.Contains(t, feed.Content, "https://me/x/Packages(Id='a',Version='1.0.0')")
	require.Contains(t, feed.Content, "proxy_endpoint="+src)

	badFeed := &nugettype.FeedEntryResponse{ID: "https://www.nuget.org/api/v2/other"}
	require.Error(t, modifyContent(badFeed, "https://me/x", "a", "1"))
}
