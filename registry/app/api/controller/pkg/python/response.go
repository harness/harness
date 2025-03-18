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

package python

import (
	"io"

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/response"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/storage"
)

var _ response.Response = (*GetMetadataResponse)(nil)
var _ response.Response = (*GetArtifactResponse)(nil)
var _ response.Response = (*PutArtifactResponse)(nil)

type GetMetadataResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
	PackageMetadata pythontype.PackageMetadata
}

func (r *GetMetadataResponse) GetErrors() []error {
	return r.Errors
}
func (r *GetMetadataResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type GetArtifactResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
	RedirectURL     string
	Body            *storage.FileReader
	ReadCloser      io.ReadCloser
}

func (r *GetArtifactResponse) GetErrors() []error {
	return r.Errors
}
func (r *GetArtifactResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type PutArtifactResponse struct {
	Sha256          string
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
}

func (r *PutArtifactResponse) GetErrors() []error {
	return r.Errors
}
func (r *PutArtifactResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}
