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

// Status types for CI.
package enum

import (
	"strings"
)

// CIStatus defines the different kinds of CI statuses for
// stages, steps and executions.
type CIStatus string

const (
	CIStatusSkipped       CIStatus = "skipped"
	CIStatusBlocked       CIStatus = "blocked"
	CIStatusDeclined      CIStatus = "declined"
	CIStatusWaitingOnDeps CIStatus = "waiting_on_dependencies"
	CIStatusPending       CIStatus = "pending"
	CIStatusRunning       CIStatus = "running"
	CIStatusSuccess       CIStatus = "success"
	CIStatusFailure       CIStatus = "failure"
	CIStatusKilled        CIStatus = "killed"
	CIStatusError         CIStatus = "error"
)

// Enum returns all possible CIStatus values.
func (CIStatus) Enum() []interface{} {
	return toInterfaceSlice(ciStatuses)
}

// Sanitize validates and returns a sanitized CIStatus value.
func (status CIStatus) Sanitize() (CIStatus, bool) {
	return Sanitize(status, GetAllCIStatuses)
}

// GetAllCIStatuses returns all possible CIStatus values and a default value.
func GetAllCIStatuses() ([]CIStatus, CIStatus) {
	return ciStatuses, CIStatusPending
}

func (status CIStatus) ConvertToCheckStatus() CheckStatus {
	if status == CIStatusPending || status == CIStatusWaitingOnDeps {
		return CheckStatusPending
	}
	if status == CIStatusSuccess || status == CIStatusSkipped {
		return CheckStatusSuccess
	}
	if status == CIStatusFailure {
		return CheckStatusFailure
	}
	if status == CIStatusRunning {
		return CheckStatusRunning
	}
	return CheckStatusError
}

// ParseCIStatus converts the status from a string to typed enum.
// If the match is not exact, will just return default error status
// instead of explicitly returning not found error.
func ParseCIStatus(status string) CIStatus {
	switch strings.ToLower(status) {
	case "skipped", "blocked", "declined", "waiting_on_dependencies",
		"pending", "running", "success", "failure", "killed", "error":
		return CIStatus(strings.ToLower(status))
	case "": // just in case status is not passed through
		return CIStatusPending
	default:
		return CIStatusError
	}
}

// IsDone returns true if in a completed state.
func (status CIStatus) IsDone() bool {
	//nolint:exhaustive
	switch status {
	case CIStatusWaitingOnDeps,
		CIStatusPending,
		CIStatusRunning,
		CIStatusBlocked:
		return false
	default:
		return true
	}
}

// IsFailed returns true if in a failed state.
func (status CIStatus) IsFailed() bool {
	return status == CIStatusFailure ||
		status == CIStatusKilled ||
		status == CIStatusError
}

// List of all CIStatus values.
var ciStatuses = sortEnum([]CIStatus{
	CIStatusSkipped,
	CIStatusBlocked,
	CIStatusDeclined,
	CIStatusWaitingOnDeps,
	CIStatusPending,
	CIStatusRunning,
	CIStatusSuccess,
	CIStatusFailure,
	CIStatusKilled,
	CIStatusError,
})
