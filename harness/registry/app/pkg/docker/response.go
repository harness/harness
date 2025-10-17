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

package docker

import (
	"io"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
)

type Response interface {
	GetErrors() []error
	SetError(error)
}

var _ Response = (*GetManifestResponse)(nil)
var _ Response = (*PutManifestResponse)(nil)
var _ Response = (*DeleteManifestResponse)(nil)

type GetManifestResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
	descriptor      manifest.Descriptor
	Manifest        manifest.Manifest
}

func (r *GetManifestResponse) GetErrors() []error {
	return r.Errors
}
func (r *GetManifestResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type PutManifestResponse struct {
	Errors []error
}

func (r *PutManifestResponse) GetErrors() []error {
	return r.Errors
}
func (r *PutManifestResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type DeleteManifestResponse struct {
	Errors []error
}

func (r *DeleteManifestResponse) GetErrors() []error {
	return r.Errors
}
func (r *DeleteManifestResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type GetBlobResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
	Body            *storage.FileReader
	Size            int64
	ReadCloser      io.ReadCloser
	RedirectURL     string
}

func (r *GetBlobResponse) GetErrors() []error {
	return r.Errors
}

func (r *GetBlobResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}
