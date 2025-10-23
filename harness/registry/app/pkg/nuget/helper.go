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
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/types"
)

// XML namespace constants.
const (
	XMLNamespaceApp                  = "http://www.w3.org/2007/app"
	XMLNamespaceAtom                 = "http://www.w3.org/2005/Atom"
	XMLNamespaceDataContract         = "http://schemas.datacontract.org/2004/07/"
	XMLNamespaceDataServicesMetadata = "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"
	XMLNamespaceDataServices         = "http://schemas.microsoft.com/ado/2007/08/dataservices"
	XMLNamespaceEdmx                 = "http://schemas.microsoft.com/ado/2007/06/edmx"
)

var semverRegexp = regexp.MustCompile("^" + SemverRegexpRaw + "$")

const SemverRegexpRaw string = `v?([0-9]+(\.[0-9]+)*?)` +
	`(-([0-9]+[0-9A-Za-z\-]*(\.[0-9A-Za-z\-]+)*)|(-([A-Za-z\-]+[0-9A-Za-z\-]*(\.[0-9A-Za-z\-]+)*)))?` +
	`(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?` +
	`?`

func buildServiceEndpoint(baseURL string) *nuget.ServiceEndpoint {
	return &nuget.ServiceEndpoint{
		Version: "3.0.0",
		Resources: []nuget.Resource{
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl",
			},
			{
				ID:   baseURL + "/package",
				Type: "PackageBaseAddress/3.0.0",
			},
			{
				ID:   baseURL,
				Type: "PackagePublish/2.0.0",
			},
			{
				ID:   baseURL + "/symbolpackage",
				Type: "SymbolPackagePublish/4.9.0",
			},
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService/3.0.0-rc",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl/3.0.0-rc",
			},
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService/3.0.0-beta",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl/3.0.0-beta",
			},
		},
	}
}

func buildServiceV2Endpoint(baseURL string) *nuget.ServiceEndpointV2 {
	return &nuget.ServiceEndpointV2{
		Base:      baseURL,
		Xmlns:     XMLNamespaceApp,
		XmlnsAtom: XMLNamespaceAtom,
		Workspace: nuget.ServiceWorkspace{
			Title: nuget.AtomTitle{
				Type: "text",
				Text: "Default",
			},
			Collection: nuget.ServiceCollection{
				Href: "Packages",
				Title: nuget.AtomTitle{
					Type: "text",
					Text: "Packages",
				},
			},
		},
	}
}

// getRegistrationIndexURL builds the registration index url.
func getRegistrationIndexURL(baseURL, id string) string {
	return fmt.Sprintf("%s/registration/%s/index.json", baseURL, id)
}

// getRegistrationLeafURL builds the registration leaf url.
func getRegistrationLeafURL(baseURL, id, version string) string {
	return fmt.Sprintf("%s/registration/%s/%s.json", baseURL, id, version)
}

// getPackageDownloadURL builds the download url.
func getPackageDownloadURL(baseURL, id, version string) string {
	return fmt.Sprintf("%s/package/%s/%s/%s.%s.nupkg", baseURL, id, version, id, version)
}

// GetPackageMetadataURL builds the package metadata url.
func getPackageMetadataURL(baseURL, id, version string) string {
	return fmt.Sprintf("%s/Packages(Id='%s',Version='%s')", baseURL, id, version)
}

func getProxyURL(baseURL, proxyEndpoint string) string {
	return fmt.Sprintf("%s?proxy_endpoint=%s", baseURL, proxyEndpoint)
}

func getInnerXMLField(baseURL, id, version string) string {
	packageMetadataURL := getPackageMetadataURL(baseURL, id, version)
	packageDownloadURL := getPackageDownloadURL(baseURL, id, version)
	return fmt.Sprintf(`<id>%s</id>
                               <content type="application/zip" src="%s"/> 
                               <link rel="edit" href="%s"/>
                               <link rel="edit" href="%s"/>`,
		packageMetadataURL, packageDownloadURL,
		packageMetadataURL, packageMetadataURL)
}

// https://learn.microsoft.com/en-us/nuget/concepts/package-versioning#normalized-version-numbers
// https://github.com/NuGet/NuGet.Client/blob/dccbd304b11103e08b97abf4cf4bcc1499d9235a/
// src/NuGet.Core/NuGet.Versioning/VersionFormatter.cs#L121.
func validateAndNormaliseVersion(v string) (string, error) {
	matches := semverRegexp.FindStringSubmatch(v)
	if matches == nil {
		return "", fmt.Errorf("malformed version: %s", v)
	}
	segmentsStr := strings.Split(matches[1], ".")
	if len(segmentsStr) > 4 {
		return "", fmt.Errorf("malformed version: %s", v)
	}
	segments := make([]int64, len(segmentsStr))
	for i, str := range segmentsStr {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return "", fmt.Errorf(
				"error parsing version: %w", err)
		}

		segments[i] = val
	}

	// Even though we could support more than three segments, if we
	// got less than three, pad it with 0s. This is to cover the basic
	// default usecase of semver, which is MAJOR.MINOR.PATCH at the minimum
	for i := len(segments); i < 3; i++ {
		//nolint:makezero
		segments = append(segments, 0)
	}

	normalizedVersion := fmt.Sprintf("%d.%d.%d", segments[0], segments[1], segments[2])
	if len(segments) > 3 && segments[3] > 0 {
		normalizedVersion = fmt.Sprintf("%s.%d", normalizedVersion, segments[3])
	}
	if len(matches) > 3 && matches[3] != "" {
		normalizedVersion = fmt.Sprintf("%s%s", normalizedVersion, matches[3])
	}
	return normalizedVersion, nil
}

func createRegistrationIndexResponse(baseURL string, info nuget.ArtifactInfo, artifacts *[]types.Artifact) (
	*nuget.RegistrationIndexResponse, error) {
	sort.Slice(*artifacts, func(i, j int) bool {
		return (*artifacts)[i].Version < (*artifacts)[j].Version
	})

	items := make([]*nuget.RegistrationIndexPageItem, 0, len(*artifacts))
	for _, p := range *artifacts {
		registrationItem, err := createRegistrationIndexPageItem(baseURL, info, &p)
		if err != nil {
			return nil, fmt.Errorf("error creating registration index page item: %w", err)
		}
		items = append(items, registrationItem)
	}

	return &nuget.RegistrationIndexResponse{
		RegistrationIndexURL: getRegistrationIndexURL(baseURL, info.Image),
		Count:                1,
		Pages: []*nuget.RegistrationIndexPageResponse{
			{
				RegistrationPageURL: getRegistrationIndexURL(baseURL, info.Image),
				Count:               len(*artifacts),
				Lower:               (*artifacts)[0].Version,
				Upper:               (*artifacts)[len(*artifacts)-1].Version,
				Items:               items,
			},
		},
	}, nil
}

func createSearchV2Response(baseURL string, artifacts *[]types.ArtifactMetadata,
	searchTerm string, limit int, offset int) (
	*nuget.FeedResponse, error) {
	links := []nuget.FeedEntryLink{
		{Rel: "self", Href: xml.CharData(baseURL)},
	}
	if artifacts == nil || len(*artifacts) == 0 {
		return &nuget.FeedResponse{
			Xmlns:   XMLNamespaceAtom,
			Base:    baseURL,
			XmlnsD:  XMLNamespaceDataServices,
			XmlnsM:  XMLNamespaceDataServicesMetadata,
			ID:      XMLNamespaceDataContract,
			Updated: time.Now(),
			Links:   links,
			Count:   0,
		}, nil
	}
	nextURL := ""
	if len(*artifacts) == limit {
		u, _ := url.Parse(baseURL)
		u = u.JoinPath("Search()")
		q := u.Query()
		q.Add("$skip", strconv.Itoa(limit+offset))
		q.Add("$top", strconv.Itoa(limit))
		if searchTerm != "" {
			q.Add("searchTerm", searchTerm)
		}
		u.RawQuery = q.Encode()
		nextURL = u.String()
	}

	if nextURL != "" {
		links = append(links, nuget.FeedEntryLink{
			Rel:  "next",
			Href: xml.CharData(nextURL),
		})
	}

	entries := make([]*nuget.FeedEntryResponse, 0, len(*artifacts))
	for _, p := range *artifacts {
		feedEntry, err := createFeedEntryResponse(baseURL,
			nuget.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: p.Name}},
			&types.Artifact{Version: p.Version, CreatedAt: p.CreatedAt, Metadata: p.Metadata, UpdatedAt: p.ModifiedAt})
		if err != nil {
			return nil, fmt.Errorf("error creating feed entry: %w", err)
		}
		entries = append(entries, feedEntry)
	}

	return &nuget.FeedResponse{
		Xmlns:   XMLNamespaceAtom,
		Base:    baseURL,
		XmlnsD:  XMLNamespaceDataServices,
		XmlnsM:  XMLNamespaceDataServicesMetadata,
		ID:      XMLNamespaceDataContract,
		Updated: time.Now(),
		Links:   links,
		Count:   int64(len(*artifacts)),
		Entries: entries,
	}, nil
}

func createSearchResponse(baseURL string, artifacts *[]types.ArtifactMetadata, totalHits int64) (
	*nuget.SearchResultResponse, error) {
	if artifacts == nil || len(*artifacts) == 0 {
		return &nuget.SearchResultResponse{
			TotalHits: totalHits,
		}, nil
	}

	var items []*nuget.SearchResult

	currentArtifact := ""
	for i := 0; i < len(*artifacts); i++ {
		if currentArtifact != (*artifacts)[i].Name {
			currentArtifact = (*artifacts)[i].Name
			searchResultItem, err := createSearchResultItem(baseURL, artifacts, currentArtifact, i)
			if err != nil {
				return nil, fmt.Errorf("error creating search result page item: %w", err)
			}
			items = append(items, searchResultItem)
		}
	}
	return &nuget.SearchResultResponse{
		TotalHits: totalHits,
		Data:      items,
	}, nil
}

func createSearchResultItem(baseURL string, artifacts *[]types.ArtifactMetadata,
	currentArtifact string, key int) (
	*nuget.SearchResult, error) {
	var items []*nuget.SearchResultVersion
	searchArtifact := &nuget.SearchResult{
		ID:                   currentArtifact,
		RegistrationIndexURL: getRegistrationIndexURL(baseURL, currentArtifact),
		Versions:             items,
	}
	for i := key; i < len(*artifacts); i++ {
		searchVersion := &nuget.SearchResultVersion{
			Version:             (*artifacts)[i].Version,
			RegistrationLeafURL: getRegistrationLeafURL(baseURL, (*artifacts)[i].Name, (*artifacts)[i].Version),
		}
		if (*artifacts)[i].Name != currentArtifact ||
			((*artifacts)[i].Name == currentArtifact && i == len(*artifacts)-1) {
			artifactMetadata := &nugetmetadata.NugetMetadata{}
			j := i - 1
			if (*artifacts)[i].Name == currentArtifact && i == len(*artifacts)-1 {
				items = append(items, searchVersion)
				j = i
			}
			err := json.Unmarshal((*artifacts)[j].Metadata, artifactMetadata)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling nuget metadata: %w", err)
			}
			searchArtifact.Description = artifactMetadata.PackageMetadata.Description
			searchArtifact.Authors = []string{artifactMetadata.PackageMetadata.Authors}
			searchArtifact.ProjectURL = artifactMetadata.PackageMetadata.ProjectURL
			searchArtifact.Version = (*artifacts)[j].Version
			searchArtifact.Versions = items
			return searchArtifact, nil
		}
		items = append(items, searchVersion)
	}
	return searchArtifact, nil
}

func modifyContent(feed *nuget.FeedEntryResponse, packageURL, pkg, version string) error {
	updatedID, err := replaceBaseWithURL(feed.ID, packageURL)
	if err != nil {
		return fmt.Errorf("error replacing base url: %w", err)
	}
	feed.Content = strings.ReplaceAll(feed.Content, feed.ID, updatedID)
	feed.Content = strings.ReplaceAll(feed.Content, feed.DownloadContent.Source,
		getProxyURL(getPackageDownloadURL(packageURL, pkg, version), feed.DownloadContent.Source))
	feed.ID = ""
	feed.DownloadContent = nil
	return nil
}

func replaceBaseWithURL(input, baseURL string) (string, error) {
	// The input is not a pure URL — it has extra after the path,
	// so we first find the index of "/Packages"
	// assuming the path always starts with "/Packages"
	const marker = "/Packages"

	idx := strings.Index(input, marker)
	if idx == -1 {
		return "", fmt.Errorf("cannot find %q in input", marker)
	}

	base := input[:idx] // the base URL part, e.g. "https://www.nuget.org/api/v2"
	path := input[idx:] // the rest starting from "/Packages..."

	// Verify base is a valid URL:
	_, err := url.ParseRequestURI(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Return with base replaced by "url"
	return baseURL + path, nil
}

func getServiceMetadataV2() *nuget.ServiceMetadataV2 {
	return &nuget.ServiceMetadataV2{
		XmlnsEdmx: XMLNamespaceEdmx,
		Version:   "1.0",
		DataServices: nuget.EdmxDataServices{
			XmlnsM:                XMLNamespaceDataServicesMetadata,
			DataServiceVersion:    "2.0",
			MaxDataServiceVersion: "2.0",
			Schema: []nuget.EdmxSchema{
				{
					Xmlns:     "http://schemas.microsoft.com/ado/2006/04/edm",
					Namespace: "NuGetGallery.OData",
					EntityType: &nuget.EdmxEntityType{
						Name:      "V2FeedPackage",
						HasStream: true,
						Keys: []nuget.EdmxPropertyRef{
							{Name: "Id"},
							{Name: "Version"},
						},
						Properties: []nuget.EdmxProperty{
							{
								Name: "Id",
								Type: "Edm.String",
							},
							{
								Name: "Version",
								Type: "Edm.String",
							},
							{
								Name:     "NormalizedVersion",
								Type:     "Edm.String",
								Nullable: true,
							},
							{
								Name:     "Authors",
								Type:     "Edm.String",
								Nullable: true,
							},
							{
								Name: "Created",
								Type: "Edm.DateTime",
							},
							{
								Name: "Dependencies",
								Type: "Edm.String",
							},
							{
								Name: "Description",
								Type: "Edm.String",
							},
							{
								Name: "DownloadCount",
								Type: "Edm.Int64",
							},
							{
								Name: "LastUpdated",
								Type: "Edm.DateTime",
							},
							{
								Name: "Published",
								Type: "Edm.DateTime",
							},
							{
								Name: "PackageSize",
								Type: "Edm.Int64",
							},
							{
								Name:     "ProjectUrl",
								Type:     "Edm.String",
								Nullable: true,
							},
							{
								Name:     "ReleaseNotes",
								Type:     "Edm.String",
								Nullable: true,
							},
							{
								Name:     "RequireLicenseAcceptance",
								Type:     "Edm.Boolean",
								Nullable: false,
							},
							{
								Name:     "Title",
								Type:     "Edm.String",
								Nullable: true,
							},
							{
								Name:     "VersionDownloadCount",
								Type:     "Edm.Int64",
								Nullable: false,
							},
						},
					},
				},
				{
					Xmlns:     "http://schemas.microsoft.com/ado/2006/04/edm",
					Namespace: "NuGetGallery",
					EntityContainer: &nuget.EdmxEntityContainer{
						Name:                     "V2FeedContext",
						IsDefaultEntityContainer: true,
						EntitySet: nuget.EdmxEntitySet{
							Name:       "Packages",
							EntityType: "NuGetGallery.OData.V2FeedPackage",
						},
						FunctionImports: []nuget.EdmxFunctionImport{
							{
								Name:       "Search",
								ReturnType: "Collection(NuGetGallery.OData.V2FeedPackage)",
								EntitySet:  "Packages",
								Parameter: []nuget.EdmxFunctionParameter{
									{
										Name: "searchTerm",
										Type: "Edm.String",
									},
								},
							},
							{
								Name:       "FindPackagesById",
								ReturnType: "Collection(NuGetGallery.OData.V2FeedPackage)",
								EntitySet:  "Packages",
								Parameter: []nuget.EdmxFunctionParameter{
									{
										Name: "id",
										Type: "Edm.String",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func createFeedResponse(baseURL string, info nuget.ArtifactInfo,
	artifacts *[]types.Artifact) (*nuget.FeedResponse, error) {
	sort.Slice(*artifacts, func(i, j int) bool {
		return (*artifacts)[i].Version < (*artifacts)[j].Version
	})

	links := []nuget.FeedEntryLink{
		{Rel: "self", Href: xml.CharData(baseURL)},
	}

	entries := make([]*nuget.FeedEntryResponse, 0, len(*artifacts))
	for _, p := range *artifacts {
		feedEntry, err := createFeedEntryResponse(baseURL, info, &p)
		if err != nil {
			return nil, fmt.Errorf("error creating feed entry: %w", err)
		}
		entries = append(entries, feedEntry)
	}

	return &nuget.FeedResponse{
		Xmlns:   XMLNamespaceAtom,
		Base:    baseURL,
		XmlnsD:  XMLNamespaceDataServices,
		XmlnsM:  XMLNamespaceDataServicesMetadata,
		ID:      XMLNamespaceDataContract,
		Updated: time.Now(),
		Links:   links,
		Count:   int64(len(*artifacts)),
		Entries: entries,
	}, nil
}

func createFeedEntryResponse(baseURL string, info nuget.ArtifactInfo, artifact *types.Artifact) (
	*nuget.FeedEntryResponse, error) {
	metadata := &nugetmetadata.NugetMetadata{}
	err := json.Unmarshal(artifact.Metadata, metadata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling nuget metadata: %w", err)
	}

	content := getInnerXMLField(baseURL, info.Image, artifact.Version)

	createdValue := nuget.TypedValue[time.Time]{
		Type:  "Edm.DateTime",
		Value: artifact.CreatedAt,
	}

	return &nuget.FeedEntryResponse{
		Xmlns:  XMLNamespaceAtom,
		Base:   baseURL,
		XmlnsD: XMLNamespaceDataServices,
		XmlnsM: XMLNamespaceDataServicesMetadata,
		Category: nuget.FeedEntryCategory{Term: "NuGetGallery.OData.V2FeedPackage",
			Scheme: "http://schemas.microsoft.com/ado/2007/08/dataservices/scheme"},
		Title:   nuget.TypedValue[string]{Type: "text", Value: info.Image},
		Updated: artifact.UpdatedAt,
		Author:  metadata.PackageMetadata.Authors,
		Content: content,
		Properties: &nuget.FeedEntryProperties{
			ID:                   info.Image,
			Version:              artifact.Version,
			NormalizedVersion:    artifact.Version,
			Authors:              metadata.PackageMetadata.Authors,
			Dependencies:         buildDependencyString(metadata),
			Description:          metadata.PackageMetadata.Description,
			VersionDownloadCount: nuget.TypedValue[int64]{Type: "Edm.Int64", Value: 0}, //todo: fix this download count
			DownloadCount:        nuget.TypedValue[int64]{Type: "Edm.Int64", Value: 0},
			PackageSize:          nuget.TypedValue[int64]{Type: "Edm.Int64", Value: metadata.Size},
			Created:              createdValue,
			LastUpdated:          createdValue,
			Published:            createdValue,
			ProjectURL:           metadata.PackageMetadata.ProjectURL,
			ReleaseNotes:         metadata.PackageMetadata.ReleaseNotes,
			RequireLicenseAcceptance: nuget.TypedValue[bool]{Type: "Edm.Boolean",
				Value: metadata.PackageMetadata.RequireLicenseAcceptance},
			Title: info.Image,
		},
	}, nil
}

func buildDependencyString(metadata *nugetmetadata.NugetMetadata) string {
	var b strings.Builder
	first := true
	if metadata.PackageMetadata.Dependencies == nil || metadata.PackageMetadata.Dependencies.Groups == nil {
		return ""
	}
	for _, deps := range metadata.PackageMetadata.Dependencies.Groups {
		for _, dep := range deps.Dependencies {
			if !first {
				b.WriteByte('|')
			}
			first = false

			b.WriteString(dep.ID)
			b.WriteByte(':')
			b.WriteString(dep.Version)
			b.WriteByte(':')
			b.WriteString(deps.TargetFramework)
		}
	}
	return b.String()
}

func createRegistrationLeafResponse(baseURL string, info nuget.ArtifactInfo,
	artifact *types.Artifact) *nuget.RegistrationLeafResponse {
	return &nuget.RegistrationLeafResponse{
		Type:                 []string{"Package", "http://schema.nuget.org/catalog#Permalink"},
		Listed:               true,
		Published:            artifact.CreatedAt,
		RegistrationLeafURL:  getRegistrationLeafURL(baseURL, info.Image, info.Version),
		PackageContentURL:    getPackageDownloadURL(baseURL, info.Image, info.Version),
		RegistrationIndexURL: getRegistrationIndexURL(baseURL, info.Image),
	}
}

func createRegistrationIndexPageItem(baseURL string, info nuget.ArtifactInfo, artifact *types.Artifact) (
	*nuget.RegistrationIndexPageItem, error) {
	metadata := &nugetmetadata.NugetMetadata{}
	err := json.Unmarshal(artifact.Metadata, metadata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling nuget metadata: %w", err)
	}

	res := &nuget.RegistrationIndexPageItem{
		RegistrationLeafURL: getRegistrationLeafURL(baseURL, info.Image, artifact.Version),
		PackageContentURL:   getPackageDownloadURL(baseURL, info.Image, artifact.Version),
		CatalogEntry: &nuget.CatalogEntry{
			CatalogLeafURL:    getRegistrationLeafURL(baseURL, info.Image, artifact.Version),
			PackageContentURL: getPackageDownloadURL(baseURL, info.Image, artifact.Version),
			ID:                info.Image,
			Version:           artifact.Version,
			Description:       metadata.PackageMetadata.Description,
			ReleaseNotes:      metadata.PackageMetadata.ReleaseNotes,
			Authors:           metadata.PackageMetadata.Authors,
			ProjectURL:        metadata.PackageMetadata.ProjectURL,
			DependencyGroups:  createDependencyGroups(metadata),
		},
	}
	dependencyGroups := createDependencyGroups(metadata)
	res.CatalogEntry.DependencyGroups = dependencyGroups
	return res, nil
}

func createDependencyGroups(metadata *nugetmetadata.NugetMetadata) []*nuget.PackageDependencyGroup {
	if metadata.PackageMetadata.Dependencies == nil {
		return nil
	}
	dependencyGroups := make([]*nuget.PackageDependencyGroup, 0,
		len(metadata.PackageMetadata.Dependencies.Groups))
	for _, group := range metadata.Metadata.PackageMetadata.Dependencies.Groups {
		deps := make([]*nuget.PackageDependency, 0, len(group.Dependencies))
		for _, dep := range group.Dependencies {
			if dep.ID == "" || dep.Version == "" {
				continue
			}
			deps = append(deps, &nuget.PackageDependency{
				ID:    dep.ID,
				Range: dep.Version,
			})
		}
		if len(deps) > 0 {
			dependencyGroups = append(dependencyGroups, &nuget.PackageDependencyGroup{
				TargetFramework: group.TargetFramework,
				Dependencies:    deps,
			})
		}
	}
	return dependencyGroups
}
