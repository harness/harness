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

package types

import (
	"github.com/harness/gitness/types/enum"
)

type AutoMerge struct {
	PullReqID    int64
	Requested    int64
	RequestedBy  int64
	MergeMethod  enum.MergeMethod
	Title        string
	Message      string
	DeleteBranch bool
}

type AutoMergeInput struct {
	Principal    Principal
	MergeMethod  enum.MergeMethod
	Title        string
	Message      string
	DeleteBranch bool
}

type AutoMergeResponse struct {
	MergeResponse *MergeResponse `json:"merge_response,omitempty"`

	Requested   int64          `json:"created,omitempty"`
	RequestedBy *PrincipalInfo `json:"requested_by,omitempty"`

	MergeMethod  enum.MergeMethod `json:"merge_method,omitempty"`
	Title        string           `json:"title,omitempty"`
	Message      string           `json:"message,omitempty"`
	DeleteBranch bool             `json:"delete_branch,omitempty"`
}
