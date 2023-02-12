// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "github.com/harness/gitness/gitrpc/rpc"

type RefType int

const (
	RefTypeUndefined RefType = iota
	RefTypeRaw
	RefTypeBranch
	RefTypeTag
	RefTypePullReqHead
	RefTypePullReqMerge
)

func RefFromRPC(t rpc.RefType) RefType {
	switch t {
	case rpc.RefType_RefRaw:
		return RefTypeRaw
	case rpc.RefType_RefBranch:
		return RefTypeBranch
	case rpc.RefType_RefTag:
		return RefTypeTag
	case rpc.RefType_RefPullReqHead:
		return RefTypePullReqHead
	case rpc.RefType_RefPullReqMerge:
		return RefTypePullReqMerge
	case rpc.RefType_Undefined:
		return RefTypeUndefined
	default:
		return RefTypeUndefined
	}
}

func RefToRPC(t RefType) rpc.RefType {
	switch t {
	case RefTypeRaw:
		return rpc.RefType_RefRaw
	case RefTypeBranch:
		return rpc.RefType_RefBranch
	case RefTypeTag:
		return rpc.RefType_RefTag
	case RefTypePullReqHead:
		return rpc.RefType_RefPullReqHead
	case RefTypePullReqMerge:
		return rpc.RefType_RefPullReqMerge
	case RefTypeUndefined:
		return rpc.RefType_Undefined
	default:
		return rpc.RefType_Undefined
	}
}
