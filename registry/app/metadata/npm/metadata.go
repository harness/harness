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

package npm

import (
	"time"

	"github.com/harness/gitness/registry/app/metadata"
)

// PackageAttachment https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md#package
// nolint:tagliatelle
type PackageAttachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
	Length      int    `json:"length"`
}

// nolint:tagliatelle
type PackageUpload struct {
	PackageMetadata
	Attachments map[string]*PackageAttachment `json:"_attachments"`
}

// nolint:tagliatelle
type PackageMetadata struct {
	ID             string                             `json:"_id"`
	Name           string                             `json:"name"`
	Description    string                             `json:"description"`
	DistTags       map[string]string                  `json:"dist-tags,omitempty"`
	Versions       map[string]*PackageMetadataVersion `json:"versions"`
	Readme         string                             `json:"readme,omitempty"`
	Maintainers    []User                             `json:"maintainers,omitempty"`
	Time           map[string]time.Time               `json:"time,omitempty"`
	Homepage       string                             `json:"homepage,omitempty"`
	Keywords       []string                           `json:"keywords,omitempty"`
	Repository     any                                `json:"repository,omitempty"`
	Author         any                                `json:"author"`
	ReadmeFilename string                             `json:"readmeFilename,omitempty"`
	Users          map[string]bool                    `json:"users,omitempty"`
	License        any                                `json:"license,omitempty"`
}

// PackageMetadataVersion documentation:
// https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md#version
// PackageMetadataVersion response:
// https://github.com/npm/registry/blob/master/docs/responses/package-metadata.md#abbreviated-version-object
// nolint:tagliatelle
type PackageMetadataVersion struct {
	ID                   string              `json:"_id"`
	Name                 string              `json:"name"`
	Version              string              `json:"version"`
	Description          any                 `json:"description"`
	Author               any                 `json:"author"`
	Homepage             any                 `json:"homepage,omitempty"`
	License              any                 `json:"license,omitempty"`
	Repository           any                 `json:"repository,omitempty"`
	Keywords             any                 `json:"keywords,omitempty"`
	Dependencies         map[string]string   `json:"dependencies,omitempty"`
	BundleDependencies   any                 `json:"bundleDependencies,omitempty"`
	DevDependencies      any                 `json:"devDependencies,omitempty"`
	PeerDependencies     any                 `json:"peerDependencies,omitempty"`
	Bin                  any                 `json:"bin,omitempty"`
	OptionalDependencies any                 `json:"optionalDependencies,omitempty"`
	Readme               string              `json:"readme,omitempty"`
	Dist                 PackageDistribution `json:"dist"`
	Maintainers          any                 `json:"maintainers,omitempty"`
}

// Repository https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md#version
// nolint:tagliatelle
type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// PackageDistribution https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md#version
// nolint:tagliatelle
type PackageDistribution struct {
	Integrity    string `json:"integrity"`
	Shasum       string `json:"shasum"`
	Tarball      string `json:"tarball"`
	FileCount    int    `json:"fileCount,omitempty"`
	UnpackedSize int    `json:"unpackedSize,omitempty"`
	NpmSignature string `json:"npm-signature,omitempty"`
}

type PackageSearch struct {
	Objects []*PackageSearchObject `json:"objects"`
	Total   int64                  `json:"total"`
}

type PackageSearchObject struct {
	Package *PackageSearchPackage `json:"package"`
}

type PackageSearchPackage struct {
	Scope       string                     `json:"scope"`
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Date        any                        `json:"date"`
	Description any                        `json:"description"`
	Author      any                        `json:"author"`
	Publisher   any                        `json:"publisher"`
	Maintainers any                        `json:"maintainers"`
	Keywords    any                        `json:"keywords,omitempty"`
	Links       *PackageSearchPackageLinks `json:"links"`
}

type PackageSearchPackageLinks struct {
	Registry   string `json:"npm"`
	Homepage   string `json:"homepage,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type User struct {
	Username string `json:"username,omitempty"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	URL      string `json:"url,omitempty"`
}

// NpmMetadata represents the metadata for a Python package.
//
//nolint:revive
type NpmMetadata struct {
	PackageMetadata
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *NpmMetadata) GetFiles() []metadata.File {
	return p.Files
}

func (p *NpmMetadata) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}

func (p *NpmMetadata) GetSize() int64 {
	return p.Size
}

func (p *NpmMetadata) UpdateSize(size int64) {
	p.Size += size
}
