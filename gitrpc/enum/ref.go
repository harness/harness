// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func (t RefType) String() string {
	switch t {
	case RefTypeRaw:
		return "raw"
	case RefTypeBranch:
		return "branch"
	case RefTypeTag:
		return "tag"
	case RefTypePullReqHead:
		return "head"
	case RefTypePullReqMerge:
		return "merge"
	case RefTypeUndefined:
		fallthrough
	default:
		return ""
	}
}
