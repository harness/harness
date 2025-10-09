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

import (
	"io"

	"github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/storage"
)

var _ response.Response = (*GetRegistryConfigResponse)(nil)
var _ response.Response = (*UploadArtifactResponse)(nil)

type BaseResponse struct {
	Error           error
	ResponseHeaders *commons.ResponseHeaders
}

func (r BaseResponse) GetError() error {
	return r.Error
}

type GetRegistryConfigResponse struct {
	BaseResponse
	Config *cargo.RegistryConfig
}

type UploadArtifactWarnings struct {
	InvalidCategories []string `json:"invalid_categories,omitempty"`
	InvalidBadges     []string `json:"invalid_badges,omitempty"`
	Other             []string `json:"other,omitempty"`
}

type UploadArtifactResponse struct {
	BaseResponse `json:"-"`
	Warnings     *UploadArtifactWarnings `json:"warnings,omitempty"`
}

type DownloadFileResponse struct {
	BaseResponse
	RedirectURL string
	Body        *storage.FileReader
	ReadCloser  io.ReadCloser
}

type GetPackageIndexResponse struct {
	DownloadFileResponse
}

type GetPackageResponse struct {
	DownloadFileResponse
}

type UpdateYankResponse struct {
	BaseResponse `json:"-"`
	Ok           bool `json:"ok"`
}

type RegeneratePackageIndexResponse struct {
	BaseResponse `json:"-"`
	Ok           bool `json:"ok"`
}

type SearchPackageResponse struct {
	BaseResponse `json:"-"`
	Crates       []SearchPackageResponseCrate  `json:"crates"`
	Metadata     SearchPackageResponseMetadata `json:"meta"`
}

type SearchPackageResponseCrate struct {
	Name        string `json:"name"`
	MaxVersion  string `json:"max_version"`
	Description string `json:"description"`
}

type SearchPackageResponseMetadata struct {
	Total int64 `json:"total"`
}
