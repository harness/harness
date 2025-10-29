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

package rpm

import "github.com/harness/gitness/registry/app/metadata"

var _ metadata.Metadata = (*RpmMetadata)(nil)

type Metadata struct {
	VersionMetadata VersionMetadata `json:"version_metadata"`
	FileMetadata    FileMetadata    `json:"file_metadata"`
}

type VersionMetadata struct {
	License     string `json:"license,omitempty"`
	ProjectURL  string `json:"project_url,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

type FileMetadata struct {
	Architecture  string `json:"architecture,omitempty"`
	Epoch         string `json:"epoch,omitempty"`
	Version       string `json:"version,omitempty"`
	Release       string `json:"release,omitempty"`
	Vendor        string `json:"vendor,omitempty"`
	Group         string `json:"group,omitempty"`
	Packager      string `json:"packager,omitempty"`
	SourceRpm     string `json:"source_rpm,omitempty"`
	BuildHost     string `json:"build_host,omitempty"`
	BuildTime     uint64 `json:"build_time,omitempty"`
	FileTime      uint64 `json:"file_time,omitempty"`
	InstalledSize uint64 `json:"installed_size,omitempty"`
	ArchiveSize   uint64 `json:"archive_size,omitempty"`

	Provides  []*Entry `json:"provide,omitempty"`
	Requires  []*Entry `json:"require,omitempty"`
	Conflicts []*Entry `json:"conflict,omitempty"`
	Obsoletes []*Entry `json:"obsolete,omitempty"`

	Files []*File `json:"files,omitempty"`

	Changelogs []*Changelog `json:"changelogs,omitempty"`
}

type Entry struct {
	Name    string `json:"name" xml:"name,attr"`
	Flags   string `json:"flags,omitempty" xml:"flags,attr,omitempty"`
	Version string `json:"version,omitempty" xml:"ver,attr,omitempty"`
	Epoch   string `json:"epoch,omitempty" xml:"epoch,attr,omitempty"`
	Release string `json:"release,omitempty" xml:"rel,attr,omitempty"`
}

type File struct {
	Path         string `json:"path" xml:",chardata"` // nolint: tagliatelle
	Type         string `json:"type,omitempty" xml:"type,attr,omitempty"`
	IsExecutable bool   `json:"is_executable" xml:"-"`
}

type Changelog struct {
	Author string `json:"author,omitempty" xml:"author,attr"`
	Date   int64  `json:"date,omitempty" xml:"date,attr"`
	Text   string `json:"text,omitempty" xml:",chardata"` // nolint: tagliatelle
}

// RpmMetadata represents the metadata for a RPM package.
//
//nolint:revive
type RpmMetadata struct {
	Metadata
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *RpmMetadata) GetSize() int64 {
	return p.Size
}

func (p *RpmMetadata) UpdateSize(size int64) {
	p.Size += size
}

func (p *RpmMetadata) GetFiles() []metadata.File {
	return p.Files
}

func (p *RpmMetadata) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}
