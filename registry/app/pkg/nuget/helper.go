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
	"fmt"

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
		Type:                 []string{"catalog:CatalogRoot", "PackageRegistration", "catalog:Permalink"},
		Count:                1,
		Pages: []*nuget.RegistrationIndexPage{
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
