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

// PullReqAttr defines pull request attribute that can be used for sorting and filtering.
type PullReqAttr int

// PullReqAttr enumeration.
const (
	PullReqAttrNone PullReqAttr = iota
	PullReqAttrCreated
	PullReqAttrUpdated
)

// ParsePullReqAttr parses the pull request attribute string
// and returns the equivalent enumeration.
func ParsePullReqAttr(s string) PullReqAttr {
	switch strings.ToLower(s) {
	case created, createdAt:
		return PullReqAttrCreated
	case updated, updatedAt:
		return PullReqAttrUpdated
	default:
		return PullReqAttrNone
	}
}

// String returns the string representation of the attribute.
func (a PullReqAttr) String() string {
	switch a {
	case PullReqAttrCreated:
		return created
	case PullReqAttrUpdated:
		return updated
	case PullReqAttrNone:
		return ""
	default:
		return undefined
	}
}
