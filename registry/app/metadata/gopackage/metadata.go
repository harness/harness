// Copyright 2023 Harness, Inc.
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

package gopackage

import "github.com/harness/gitness/registry/app/metadata"

// nolint:tagliatelle
type VersionMetadata struct {
	Name         string       `json:"Name"`
	Version      string       `json:"Version"`
	Time         string       `json:"Time"`
	Origin       Origin       `json:"Origin,omitempty"`
	Dependencies []Dependency `json:"Dependencies,omitempty"`
}

// nolint:tagliatelle
type Origin struct {
	VCS  string `json:"VCS,omitempty"`
	URL  string `json:"URL,omitempty"`
	Ref  string `json:"Ref,omitempty"`
	Hash string `json:"Hash,omitempty"`
}

// nolint:tagliatelle
type Dependency struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
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
