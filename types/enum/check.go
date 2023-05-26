// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// CheckStatus defines status check status.
type CheckStatus string

func (CheckStatus) Enum() []interface{}                 { return toInterfaceSlice(checkStatuses) }
func (s CheckStatus) Sanitize() (CheckStatus, bool)     { return Sanitize(s, GetAllCheckStatuses) }
func GetAllCheckStatuses() ([]CheckStatus, CheckStatus) { return checkStatuses, "" }

// PullReqState enumeration.
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
