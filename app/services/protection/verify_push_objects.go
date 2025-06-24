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

package protection

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	PushObjectsVerifyInput struct {
		ResolveUserGroupID func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor              *types.Principal
		IsRepoOwner        bool
		RepoID             int64
	}

	PushObjectsVerifyOutput struct {
		FileSizeLimit           int64
		PrincipalCommitterMatch bool
	}

	PushObjectsVerifier interface {
		PushObjectsVerify(
			ctx context.Context,
			in PushObjectsVerifyInput,
		) (PushObjectsVerifyOutput, []types.RuleViolations, error)
	}

	DefPushObjects struct {
		FileSizeLimit           int64 `json:"file_size_limit"`
		PrincipalCommitterMatch bool  `json:"principal_committer_match"`
	}
)

func (v *DefPushObjects) PushObjectsVerify(
	_ context.Context,
	_ PushObjectsVerifyInput,
) (PushObjectsVerifyOutput, []types.RuleViolations, error) {
	return PushObjectsVerifyOutput{
		FileSizeLimit:           v.FileSizeLimit,
		PrincipalCommitterMatch: v.PrincipalCommitterMatch,
	}, nil, nil
}
