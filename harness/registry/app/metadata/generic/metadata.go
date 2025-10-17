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

package generic

import "github.com/harness/gitness/registry/app/metadata"

var _ metadata.Metadata = (*GenericMetadata)(nil)

// GenericMetadata represents the metadata for a Generic package.
//
//nolint:revive
type GenericMetadata struct {
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *GenericMetadata) GetFiles() []metadata.File {
	return p.Files
}

func (p *GenericMetadata) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}
func (p *GenericMetadata) GetSize() int64 {
	return p.Size
}

func (p *GenericMetadata) UpdateSize(size int64) {
	p.Size += size
}
