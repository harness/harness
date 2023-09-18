// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	case "skipped", "blocked", "declined", "waiting_on_dependencies", "pending", "running", "success", "failure", "killed", "error":
		return CIStatus(strings.ToLower(status))
	case "": // just in case status is not passed through
		return CIStatusPending
	default:
		return CIStatusError
	}
}

// IsDone returns true if in a completed state.
func (status CIStatus) IsDone() bool {
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
	switch status {
	case CIStatusFailure,
		CIStatusKilled,
		CIStatusError:
		return true
	default:
		return false
	}
}
