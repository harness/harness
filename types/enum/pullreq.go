// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// PullReqState defines pull request state.
type PullReqState string

// PullReqState enumeration.
const (
	PullReqStateOpen     PullReqState = "open"
	PullReqStateMerged   PullReqState = "merged"
	PullReqStateClosed   PullReqState = "closed"
	PullReqStateRejected PullReqState = "rejected"
)

// PullReqSort defines pull request attribute that can be used for sorting.
type PullReqSort int

// PullReqAttr enumeration.
const (
	PullReqSortNone PullReqSort = iota
	PullReqSortNumber
	PullReqSortCreated
	PullReqSortUpdated
)

// ParsePullReqSort parses the pull request attribute string
// and returns the equivalent enumeration.
func ParsePullReqSort(s string) PullReqSort {
	switch strings.ToLower(s) {
	case number:
		return PullReqSortNumber
	case created, createdAt:
		return PullReqSortCreated
	case updated, updatedAt:
		return PullReqSortUpdated
	default:
		return PullReqSortNone
	}
}

// String returns the string representation of the attribute.
func (a PullReqSort) String() string {
	switch a {
	case PullReqSortNumber:
		return number
	case PullReqSortCreated:
		return created
	case PullReqSortUpdated:
		return updated
	case PullReqSortNone:
		return ""
	default:
		return undefined
	}
}
