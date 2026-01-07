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

package lfs

import (
	"time"

	"github.com/harness/gitness/types/enum"
)

type Reference struct {
	Name string `json:"name"`
}

// Pointer contains LFS pointer data.
type Pointer struct {
	OId  string `json:"oid"`
	Size int64  `json:"size"`
}

type TransferInput struct {
	Operation enum.GitLFSOperationType  `json:"operation"`
	Transfers []enum.GitLFSTransferType `json:"transfers,omitempty"`
	Ref       *Reference                `json:"ref,omitempty"`
	Objects   []Pointer                 `json:"objects"`
	HashAlgo  string                    `json:"hash_algo,omitempty"`
}

// ObjectError defines the JSON structure returned to the client in case of an error.
type ObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Action provides a structure with information about next actions fo the object.
type Action struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn *time.Duration    `json:"expires_in,omitempty"`
}

// ObjectResponse is object metadata as seen by clients of the LFS server.
type ObjectResponse struct {
	Pointer
	Authenticated *bool             `json:"authenticated,omitempty"`
	Actions       map[string]Action `json:"actions"`
	Error         *ObjectError      `json:"error,omitempty"`
}

type TransferOutput struct {
	Transfer enum.GitLFSTransferType `json:"transfer"`
	Objects  []ObjectResponse        `json:"objects"`
}

type AuthenticateResponse struct {
	Header    map[string]string `json:"header"`
	HRef      string            `json:"href"`
	ExpiresIn time.Duration     `json:"expires_in"`
}
