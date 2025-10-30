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
	"io"

	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
)

type GetMetadataResponse struct {
	BaseResponse
	PackageMetadata npm2.PackageMetadata
}

func (r *GetMetadataResponse) GetError() error {
	return r.Error
}

type ListTagResponse struct {
	Tags map[string]string
	BaseResponse
}

func (r *ListTagResponse) GetError() error {
	return r.Error
}

type BaseResponse struct {
	Error           error
	ResponseHeaders *commons.ResponseHeaders
}

type GetArtifactResponse struct {
	BaseResponse
	RedirectURL string
	Body        *storage.FileReader
	ReadCloser  io.ReadCloser
}

func (r *GetArtifactResponse) GetError() error {
	return r.Error
}

type PutArtifactResponse struct {
	Sha256 string
	BaseResponse
}

func (r *PutArtifactResponse) GetError() error {
	return r.Error
}

type HeadMetadataResponse struct {
	BaseResponse
	Exists bool
}

func (r *HeadMetadataResponse) GetError() error {
	return r.Error
}

type DeleteEntityResponse struct {
	Error error
}

func (r *DeleteEntityResponse) GetError() error {
	return r.Error
}

type SearchArtifactResponse struct {
	BaseResponse
	Artifacts *npm2.PackageSearch
}

func (r *SearchArtifactResponse) GetError() error {
	return r.Error
}
