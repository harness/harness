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

package cargo

import "github.com/harness/gitness/registry/app/metadata"

type RegistryConfig struct {
	DownloadURL  string `json:"dl"`
	APIURL       string `json:"api"`
	AuthRequired bool   `json:"auth-required"` //nolint:tagliatelle
}

type DependencyKindType string

var (
	DependencyKindTypeDev    DependencyKindType = "dev"
	DependencyKindTypeBuild  DependencyKindType = "build"
	DependencyKindTypeNormal DependencyKindType = "normal"
)

type Dependency struct {
	Name               string             `json:"name"`
	Features           []string           `json:"features,omitempty"`
	IsOptional         bool               `json:"optional,omitempty"`
	DefaultFeatures    bool               `json:"default_features,omitempty"`
	Target             string             `json:"target,omitempty"`
	Kind               DependencyKindType `json:"kind,omitempty"`
	Registry           string             `json:"registry,omitempty"`
	ExplicitNameInToml string             `json:"explicit_name_in_toml,omitempty"`
}

type VersionDependency struct {
	Dependency
	VersionRequired string `json:"version_req,omitempty"`
}

type IndexDependency struct {
	Dependency
	VersionRequired string `json:"req,omitempty"`
}

type VersionMetadata struct {
	Name             string              `json:"name"`
	Version          string              `json:"vers"`
	ReadmeFile       string              `json:"readme_file,omitempty"`
	Keywords         []string            `json:"keywords,omitempty"`
	License          string              `json:"license,omitempty"`
	LicenseFile      string              `json:"license_file,omitempty"`
	Links            []string            `json:"links,omitempty"`
	Dependencies     []VersionDependency `json:"deps,omitempty"`
	Authors          []string            `json:"authors,omitempty"`
	Description      string              `json:"description,omitempty"`
	Categories       []string            `json:"categories,omitempty"`
	RepositoryURL    string              `json:"repository,omitempty"`
	Badges           map[string]any      `json:"badges,omitempty"`
	Features         map[string][]string `json:"features,omitempty"`
	DocumentationURL string              `json:"documentation,omitempty"`
	HomepageURL      string              `json:"homepage,omitempty"`
	Readme           string              `json:"readme,omitempty"`
	RustVersion      string              `json:"rust_version,omitempty"`
	Yanked           bool                `json:"yanked"`
}

type IndexMetadata struct {
	Name         string              `json:"name"`
	Version      string              `json:"vers"`
	Checksum     string              `json:"cksum"`
	Features     map[string][]string `json:"features"`
	Dependencies []IndexDependency   `json:"deps"`
	Yanked       bool                `json:"yanked"`
}

type VersionMetadataDB struct {
	VersionMetadata
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *VersionMetadataDB) GetFiles() []metadata.File {
	return p.Files
}

func (p *VersionMetadataDB) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}

func (p *VersionMetadataDB) GetSize() int64 {
	return p.Size
}

func (p *VersionMetadataDB) UpdateSize(size int64) {
	p.Size += size
}
