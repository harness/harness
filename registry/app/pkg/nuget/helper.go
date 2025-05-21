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
	"strings"
	"time"

	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/types"
)

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
		Xmlns:     "http://www.w3.org/2007/app",
		XmlnsAtom: "http://www.w3.org/2005/Atom",
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

// GetPackageMetadataURL builds the package metadata url
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

func createRegistrationIndexResponse(baseURL string, info nuget.ArtifactInfo, artifacts *[]types.Artifact) (
	*nuget.RegistrationIndexResponse, error) {
	//todo: sort in ascending order

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

func createFeedResponse(baseURL string, info nuget.ArtifactInfo, artifacts *[]types.Artifact) (
	*nuget.FeedResponse, error) {
	//todo: sort in ascending order

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
		Xmlns:   "http://www.w3.org/2005/Atom",
		Base:    baseURL,
		XmlnsD:  "http://schemas.microsoft.com/ado/2007/08/dataservices",
		XmlnsM:  "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata",
		ID:      "http://schemas.datacontract.org/2004/07/",
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
		Xmlns:  "http://www.w3.org/2005/Atom",
		Base:   baseURL,
		XmlnsD: "http://schemas.microsoft.com/ado/2007/08/dataservices",
		XmlnsM: "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata",
		Category: nuget.FeedEntryCategory{Term: "NuGetGallery.OData.V2FeedPackage",
			Scheme: "http://schemas.microsoft.com/ado/2007/08/dataservices/scheme"},
		Title:   nuget.TypedValue[string]{Type: "text", Value: info.Image},
		Updated: artifact.UpdatedAt,
		Author:  metadata.PackageMetadata.Authors,
		Content: content,
		Properties: &nuget.FeedEntryProperties{
			Id:                       info.Image,
			Version:                  artifact.Version,
			NormalizedVersion:        artifact.Version,
			Authors:                  metadata.PackageMetadata.Authors,
			Dependencies:             buildDependencyString(metadata),
			Description:              metadata.PackageMetadata.Description,
			VersionDownloadCount:     nuget.TypedValue[int64]{Type: "Edm.Int64", Value: 0}, //todo: fix this download count
			DownloadCount:            nuget.TypedValue[int64]{Type: "Edm.Int64", Value: 0},
			PackageSize:              nuget.TypedValue[int64]{Type: "Edm.Int64", Value: metadata.Size},
			Created:                  createdValue,
			LastUpdated:              createdValue,
			Published:                createdValue,
			ProjectURL:               metadata.PackageMetadata.ProjectURL,
			ReleaseNotes:             metadata.PackageMetadata.ReleaseNotes,
			RequireLicenseAcceptance: nuget.TypedValue[bool]{Type: "Edm.Boolean", Value: metadata.PackageMetadata.RequireLicenseAcceptance},
			Title:                    info.Image,
		},
	}, nil
}

func buildDependencyString(metadata *nugetmetadata.NugetMetadata) string {
	var b strings.Builder
	first := true
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
