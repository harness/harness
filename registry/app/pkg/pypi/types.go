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

// Metadata represents the metadata for a PyPI package.
type Metadata struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description"`
	Author         string   `json:"author"`
	AuthorEmail    string   `json:"author_email,omitempty"`
	License        string   `json:"license"`
	Keywords       []string `json:"keywords,omitempty"`
	Platform       string   `json:"platform,omitempty"`
	RequiresPython string   `json:"requires_python,omitempty"`
	Dependencies   []string `json:"dependencies,omitempty"`
}

// PackageFile represents a PyPI package file.
type PackageFile struct {
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	Size          int64  `json:"size"`
	MD5           string `json:"md5_digest"`
	SHA256        string `json:"sha256_digest"`
	PackageType   string `json:"package_type"` // e.g., "sdist", "bdist_wheel"
	PythonVersion string `json:"python_version"`
	UploadTime    int64  `json:"upload_time_ms"`
}
