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

package huggingface

import (
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/response"
	huggingfacetype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/storage"
)

var _ response.Response = (*ValidateYamlResponse)(nil)
var _ response.Response = (*PreUploadResponse)(nil)
var _ response.Response = (*RevisionInfoResponse)(nil)
var _ response.Response = (*LfsInfoResponse)(nil)
var _ response.Response = (*LfsVerifyResponse)(nil)
var _ response.Response = (*LfsUploadResponse)(nil)
var _ response.Response = (*HeadFileResponse)(nil)
var _ response.Response = (*DownloadFileResponse)(nil)

// Response is the base response interface.
type BaseResponse struct {
	Error           error
	ResponseHeaders *commons.ResponseHeaders
}

func (r BaseResponse) GetError() error {
	return r.Error
}

type ValidateYamlResponse struct {
	BaseResponse
	Response *huggingfacetype.ValidateYamlResponse
}

type PreUploadResponse struct {
	BaseResponse
	Response *huggingfacetype.PreUploadResponse
}

// RevisionInfoResponse represents a response for revision info.
type RevisionInfoResponse struct {
	BaseResponse
	Response *huggingfacetype.RevisionInfoResponse
}

type LfsInfoResponse struct {
	BaseResponse
	Response *huggingfacetype.LfsInfoResponse
}

type LfsUploadResponse struct {
	BaseResponse
	Response *huggingfacetype.LfsUploadResponse
}

type LfsVerifyResponse struct {
	BaseResponse
	Response *huggingfacetype.LfsVerifyResponse
}

type CommitRevisionResponse struct {
	BaseResponse
	Response *huggingfacetype.CommitRevisionResponse
}

type HeadFileResponse struct {
	BaseResponse
}

type DownloadFileResponse struct {
	BaseResponse
	RedirectURL string
	Body        *storage.FileReader
}
