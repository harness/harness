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

package maven

import (
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
)

type Response interface {
	GetErrors() []error
	SetError(error)
}

var _ Response = (*HeadArtifactResponse)(nil)
var _ Response = (*GetArtifactResponse)(nil)
var _ Response = (*PutArtifactResponse)(nil)

type HeadArtifactResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
}

func (r *HeadArtifactResponse) GetErrors() []error {
	return r.Errors
}
func (r *HeadArtifactResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type GetArtifactResponse struct {
	Errors          []error
	ResponseHeaders *commons.ResponseHeaders
	RedirectURL     string
	Body            *storage.FileReader
}

func (r *GetArtifactResponse) GetErrors() []error {
	return r.Errors
}
func (r *GetArtifactResponse) SetError(err error) {
	r.Errors = make([]error, 1)
	r.Errors[0] = err
}

type PutArtifactResponse struct {
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
