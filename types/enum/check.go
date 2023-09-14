// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

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
