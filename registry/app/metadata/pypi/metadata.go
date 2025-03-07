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

package pypi

// Metadata Source: https://github.com/pypa/twine/blob/main/twine/package.py
type Metadata struct {
	// Metadata 1.0
	MetadataVersion string   `json:"metadata_version,omitempty"`
	Name            string   `json:"name,omitempty"`
	Version         string   `json:"version,omitempty"`
	Platform        []string `json:"platform,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Description     string   `json:"description,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
	HomePage        string   `json:"home_page,omitempty"`
	Author          string   `json:"author,omitempty"`
	AuthorEmail     string   `json:"author_email,omitempty"`
	License         string   `json:"license,omitempty"`

	// Metadata 1.1
	SupportedPlatform []string `json:"supported_platform,omitempty"`
	DownloadURL       string   `json:"download_url,omitempty"`
	Classifiers       []string `json:"classifiers,omitempty"`
	Requires          []string `json:"requires,omitempty"`
	Provides          []string `json:"provides,omitempty"`
	Obsoletes         []string `json:"obsoletes,omitempty"`

	// Metadata 1.2
	Maintainer       string            `json:"maintainer,omitempty"`
	MaintainerEmail  string            `json:"maintainer_email,omitempty"`
	RequiresDist     []string          `json:"requires_dist,omitempty"`
	ProvidesDist     []string          `json:"provides_dist,omitempty"`
	ObsoletesDist    []string          `json:"obsoletes_dist,omitempty"`
	RequiresPython   string            `json:"requires_python,omitempty"`
	RequiresExternal []string          `json:"requires_external,omitempty"`
	ProjectURLs      map[string]string `json:"project_urls,omitempty"`

	// Metadata 2.1
	DescriptionContentType string   `json:"description_content_type,omitempty"`
	ProvidesExtra          []string `json:"provides_extra,omitempty"`

	// Metadata 2.2
	Dynamic []string `json:"dynamic,omitempty"`

	// Metadata 2.4
	LicenseExpression string   `json:"license_expression,omitempty"`
	LicenseFile       []string `json:"license_file,omitempty"`

	// Additional metadata
	Comment         string   `json:"comment,omitempty"`
	PyVersion       string   `json:"pyversion,omitempty"`
	FileType        string   `json:"filetype,omitempty"`
	GPGSignature    []string `json:"gpg_signature,omitempty"`
	Attestations    string   `json:"attestations,omitempty"`
	MD5Digest       string   `json:"md5_digest,omitempty"`
	SHA256Digest    string   `json:"sha256_digest,omitempty"`
	Blake2256Digest string   `json:"blake2_256_digest,omitempty"`

	// Legacy fields kept for compatibility
	LongDescription string   `json:"long_description,omitempty"`
	ProjectURL      string   `json:"project_url,omitempty"`
	Dependencies    []string `json:"dependencies,omitempty"`
}
