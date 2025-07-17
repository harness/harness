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

package gopackage

import (
	"io"

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/storage"
)

var _ response.Response = (*DownloadFileResponse)(nil)

type BaseResponse struct {
	Error           error
	ResponseHeaders *commons.ResponseHeaders
}

func (r BaseResponse) GetError() error {
	return r.Error
}

type DownloadFileResponse struct {
	BaseResponse
	RedirectURL string
	Body        *storage.FileReader
	ReadCloser  io.ReadCloser
}
