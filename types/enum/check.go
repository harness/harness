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

package enum

import "golang.org/x/exp/slices"

// CheckStatus defines status check status.
type CheckStatus string

func (CheckStatus) Enum() []interface{}                 { return toInterfaceSlice(checkStatuses) }
func (s CheckStatus) Sanitize() (CheckStatus, bool)     { return Sanitize(s, GetAllCheckStatuses) }
func GetAllCheckStatuses() ([]CheckStatus, CheckStatus) { return checkStatuses, "" }

// CheckStatus enumeration.
const (
	CheckStatusPending CheckStatus = "pending"
	CheckStatusRunning CheckStatus = "running"
	CheckStatusSuccess CheckStatus = "success"
	CheckStatusFailure CheckStatus = "failure"
	CheckStatusError   CheckStatus = "error"
)

var checkStatuses = sortEnum([]CheckStatus{
	CheckStatusPending,
	CheckStatusRunning,
	CheckStatusSuccess,
	CheckStatusFailure,
	CheckStatusError,
})

var terminalCheckStatuses = []CheckStatus{CheckStatusFailure, CheckStatusSuccess, CheckStatusError}

// CheckPayloadKind defines status payload type.
type CheckPayloadKind string

func (CheckPayloadKind) Enum() []interface{} { return toInterfaceSlice(checkPayloadTypes) }
func (s CheckPayloadKind) Sanitize() (CheckPayloadKind, bool) {
	return Sanitize(s, GetAllCheckPayloadTypes)
}
func GetAllCheckPayloadTypes() ([]CheckPayloadKind, CheckPayloadKind) {
	return checkPayloadTypes, CheckPayloadKindEmpty
}

// CheckPayloadKind enumeration.
const (
	CheckPayloadKindEmpty    CheckPayloadKind = ""
	CheckPayloadKindRaw      CheckPayloadKind = "raw"
	CheckPayloadKindMarkdown CheckPayloadKind = "markdown"
	CheckPayloadKindPipeline CheckPayloadKind = "pipeline"
)

var checkPayloadTypes = sortEnum([]CheckPayloadKind{
	CheckPayloadKindEmpty,
	CheckPayloadKindRaw,
	CheckPayloadKindMarkdown,
	CheckPayloadKindPipeline,
})

func (s CheckStatus) IsCompleted() bool {
	return slices.Contains(terminalCheckStatuses, s)
}
