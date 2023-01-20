// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "github.com/harness/gitness/gitrpc/rpc"

type RefType int

const (
	RefTypeRaw RefType = iota
	RefTypeBranch
	RefTypeTag
	RefTypePullReqHead
	RefTypePullReqMerge
)

func RefFromRPC(t rpc.RefType) (RefType, bool) {
	switch t {
	case rpc.RefType_RefRaw:
		return RefTypeRaw, true
	case rpc.RefType_RefBranch:
		return RefTypeBranch, true
	case rpc.RefType_RefTag:
		return RefTypeTag, true
	case rpc.RefType_RefPullReqHead:
		return RefTypePullReqHead, true
	case rpc.RefType_RefPullReqMerge:
		return RefTypePullReqMerge, true
	default:
		return 0, false
	}
}

func RefToRPC(t RefType) (rpc.RefType, bool) {
	switch t {
	case RefTypeRaw:
		return rpc.RefType_RefRaw, true
	case RefTypeBranch:
		return rpc.RefType_RefBranch, true
	case RefTypeTag:
		return rpc.RefType_RefTag, true
	case RefTypePullReqHead:
		return rpc.RefType_RefPullReqHead, true
	case RefTypePullReqMerge:
		return rpc.RefType_RefPullReqMerge, true
	default:
		return 0, false
	}
}
