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

package huggingface

import "github.com/harness/gitness/registry/app/metadata"

// CardData represents the card data for a HuggingFace model.
type CardData struct {
	Language []string `json:"language,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	License  string   `json:"license,omitempty"`
}

// Sibling represents a file in a HuggingFace model.
type Sibling struct {
	RFilename string `json:"rfilename"`
}

var _ metadata.Metadata = (*HuggingFaceMetadata)(nil)

type Metadata struct {
	ID           string    `json:"id"`
	ModelID      string    `json:"modelId,omitempty"`
	SHA          string    `json:"sha,omitempty"`
	Downloads    int64     `json:"downloads,omitempty"`
	Likes        int64     `json:"likes,omitempty"`
	LibraryName  string    `json:"libraryName,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	CardData     *CardData `json:"cardData,omitempty"`
	Siblings     []Sibling `json:"siblings,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	Private      bool      `json:"private,omitempty"`
	Readme       string    `json:"readme,omitempty"`
}

//nolint:revive
type HuggingFaceMetadata struct {
	Metadata
	Files     []metadata.File `json:"files"`
	FileCount int64           `json:"file_count"`
	Size      int64           `json:"size"`
}

func (p *HuggingFaceMetadata) GetFiles() []metadata.File {
	return p.Files
}

func (p *HuggingFaceMetadata) SetFiles(files []metadata.File) {
	p.Files = files
	p.FileCount = int64(len(files))
}
func (p *HuggingFaceMetadata) GetSize() int64 {
	return p.Size
}

func (p *HuggingFaceMetadata) UpdateSize(size int64) {
	p.Size += size
}
