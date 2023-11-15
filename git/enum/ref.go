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

type RefType int

const (
	RefTypeUndefined RefType = iota
	RefTypeRaw
	RefTypeBranch
	RefTypeTag
	RefTypePullReqHead
	RefTypePullReqMerge
)

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
