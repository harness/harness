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
	"time"

	"github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
)

type ArtifactInfo struct {
	pkg.ArtifactInfo
	Version  string
	Filename string
	Metadata nuget.Metadata
}

// BaseArtifactInfo implements pkg.PackageArtifactInfo interface.
func (a ArtifactInfo) BaseArtifactInfo() pkg.ArtifactInfo {
	return a.ArtifactInfo
}

func (a ArtifactInfo) GetImageVersion() (exists bool, imageVersion string) {
	if a.Image != "" && a.Version != "" {
		return true, pkg.JoinWithSeparator(":", a.Image, a.Version)
	}
	return false, ""
}

func (a ArtifactInfo) GetVersion() string {
	return a.Version
}

type File struct {
	FileURL string
	Name    string
}

type PackageMetadata struct {
	Name  string
	Files []File
}

type ServiceEndpoint struct {
	Version   string     `json:"version"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	//nolint:revive
	ID   string `json:"@id"`
	Type string `json:"@type"`
}

type PackageVersion struct {
	Versions []string `json:"versions"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#response
type RegistrationIndexResponse struct {
	RegistrationIndexURL string                   `json:"@id"`
	Type                 []string                 `json:"@type"`
	Count                int                      `json:"count"`
	Pages                []*RegistrationIndexPage `json:"items"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-page-object
type RegistrationIndexPage struct {
	RegistrationPageURL string                       `json:"@id"`
	Lower               string                       `json:"lower"`
	Upper               string                       `json:"upper"`
	Count               int                          `json:"count"`
	Items               []*RegistrationIndexPageItem `json:"items"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-leaf-object-in-a-page
type RegistrationIndexPageItem struct {
	RegistrationLeafURL string `json:"@id"`
	//nolint: tagliatelle
	PackageContentURL string `json:"packageContent"`
	//nolint: tagliatelle
	CatalogEntry *CatalogEntry `json:"catalogEntry"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#registration-leaf
type RegistrationLeafResponse struct {
	RegistrationLeafURL string   `json:"@id"`
	Type                []string `json:"@type"`
	Listed              bool     `json:"listed"`
	//nolint: tagliatelle
	PackageContentURL    string    `json:"packageContent"`
	Published            time.Time `json:"published"`
	RegistrationIndexURL string    `json:"registration"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#catalog-entry
type CatalogEntry struct {
	CatalogLeafURL string `json:"@id"`
	//nolint: tagliatelle
	PackageContentURL string `json:"packageContent"`
	ID                string `json:"id"`
	Version           string `json:"version"`
	Description       string `json:"description"`
	//nolint: tagliatelle
	ReleaseNotes string `json:"releaseNotes"`
	Authors      string `json:"authors"`
	//nolint: tagliatelle
	RequireLicenseAcceptance bool `json:"requireLicenseAcceptance"`
	//nolint: tagliatelle
	ProjectURL string `json:"projectURL"`
	//nolint: tagliatelle
	DependencyGroups []*PackageDependencyGroup `json:"dependencyGroups,omitempty"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#package-dependency-group
type PackageDependencyGroup struct {
	//nolint: tagliatelle
	TargetFramework string               `json:"targetFramework"`
	Dependencies    []*PackageDependency `json:"dependencies"`
}

// https://docs.microsoft.com/en-us/nuget/api/registration-base-url-resource#package-dependency
type PackageDependency struct {
	ID    string `json:"id"`
	Range string `json:"range"`
}
