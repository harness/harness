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
	"github.com/harness/gitness/registry/app/metadata"
)

var _ metadata.Metadata = (*NugetMetadata)(nil)

//nolint:revive
type NugetMetadata struct {
	Metadata
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *NugetMetadata) GetFiles() []metadata.File {
	return p.Files
}

func (p *NugetMetadata) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}

func (p *NugetMetadata) GetSize() int64 {
	return p.Size
}

func (p *NugetMetadata) UpdateSize(size int64) {
	p.Size += size
}

// Package represents the entire NuGet package.
type Metadata struct {
	PackageMetadata PackageMetadata `json:"metadata" xml:"metadata"`
	Files           *FilesWrapper   `json:"files,omitempty" xml:"files,omitempty"`
}

// PackageMetadata represents the metadata of a NuGet package.
type PackageMetadata struct {
	ID                       string               `json:"id" xml:"id"`
	Version                  string               `json:"version" xml:"version"`
	Title                    string               `json:"title,omitempty" xml:"title,omitempty"`
	Authors                  string               `json:"authors" xml:"authors"`
	Owners                   string               `json:"owners,omitempty" xml:"owners,omitempty"`
	LicenseURL               string               `json:"licenseUrl,omitempty" xml:"licenseUrl,omitempty"`
	ProjectURL               string               `json:"projectUrl,omitempty" xml:"projectUrl,omitempty"`
	IconURL                  string               `json:"iconUrl,omitempty" xml:"iconUrl,omitempty"`
	RequireLicenseAcceptance bool                 `json:"requireLicenseAcceptance,omitempty" xml:"requireLicenseAcceptance,omitempty"`
	DevelopmentDependency    bool                 `json:"developmentDependency,omitempty" xml:"developmentDependency,omitempty"`
	Description              string               `json:"description" xml:"description"`
	Summary                  string               `json:"summary,omitempty" xml:"summary,omitempty"`
	ReleaseNotes             string               `json:"releaseNotes,omitempty" xml:"releaseNotes,omitempty"`
	Copyright                string               `json:"copyright,omitempty" xml:"copyright,omitempty"`
	Language                 string               `json:"language,omitempty" xml:"language,omitempty"`
	Tags                     string               `json:"tags,omitempty" xml:"tags,omitempty"`
	Serviceable              bool                 `json:"serviceable,omitempty" xml:"serviceable,omitempty"`
	Icon                     string               `json:"icon,omitempty" xml:"icon,omitempty"`
	Readme                   string               `json:"readme,omitempty" xml:"readme,omitempty"`
	Repository               *Repository          `json:"repository,omitempty" xml:"repository,omitempty"`
	License                  *License             `json:"license,omitempty" xml:"license,omitempty"`
	PackageTypes             []PackageType        `json:"packageTypes,omitempty" xml:"packageType"`
	Dependencies             *DependenciesWrapper `json:"dependencies,omitempty" xml:"dependencies,omitempty"`
	MinClientVersion         string               `json:"minClientVersion,omitempty" xml:"minClientVersion,attr"`
}

// Dependency represents a package dependency.
type Dependency struct {
	ID      string `json:"id,omitempty" xml:"id,attr"`
	Version string `json:"version,omitempty" xml:"version,attr"`
	Include string `json:"include,omitempty" xml:"include,attr"`
	Exclude string `json:"exclude,omitempty" xml:"exclude,attr"`
}

// DependencyGroup represents a group of dependencies.
type DependencyGroup struct {
	TargetFramework string       `json:"targetFramework,omitempty" xml:"targetFramework,attr"`
	Dependencies    []Dependency `json:"dependencies,omitempty" xml:"dependency"`
}

// PackageType represents a package type.
type PackageType struct {
	Name    string `json:"name" xml:"name,attr"`
	Version string `json:"version,omitempty" xml:"version,attr"`
}

// Repository represents the repository information.
type Repository struct {
	Type   string `json:"type,omitempty" xml:"type,attr"`
	URL    string `json:"url,omitempty" xml:"url,attr"`
	Branch string `json:"branch,omitempty" xml:"branch,attr"`
	Commit string `json:"commit,omitempty" xml:"commit,attr"`
}

// License represents a package license.
type License struct {
	Type    string `json:"type" xml:"type,attr"`
	Version string `json:"version,omitempty" xml:"version,attr"`
	//nolint:staticcheck
	Text string `json:",chardata" xml:",chardata"`
}

// DependenciesWrapper represents the `<dependencies>` section.
type DependenciesWrapper struct {
	Dependencies []Dependency      `json:"dependencies,omitempty" xml:"dependency"`
	Groups       []DependencyGroup `json:"groups,omitempty" xml:"group"`
}

// File represents an individual file in the `<files>` section.
type File struct {
	Src     string `json:"src" xml:"src,attr"`
	Target  string `json:"target,omitempty" xml:"target,attr"`
	Exclude string `json:"exclude,omitempty" xml:"exclude,attr"`
}

// FilesWrapper represents the `<files>` section.
type FilesWrapper struct {
	Files []File `json:"files,omitempty" xml:"file"`
}
