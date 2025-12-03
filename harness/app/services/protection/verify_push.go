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

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

const (
	codePushFileSizeLimit           = "push.file.size.limit"
	codePushPrincipalCommitterMatch = "push.principal.committer.match"
	codeSecretScanningEnabled       = "push.secret.scanning.enabled"
)

type (
	PushVerifyInput struct {
		ResolveUserGroupID func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor              *types.Principal
		IsRepoOwner        bool
		RepoID             int64
	}

	PushViolationsInput struct {
		ResolveUserGroupID      func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor                   *types.Principal
		IsRepoOwner             bool
		Protections             map[int64]PushProtection
		FileSizeLimit           int64
		FindOversizeFilesOutput *git.FindOversizeFilesOutput
		PrincipalCommitterMatch bool
		CommitterMismatchCount  int64
		SecretScanningEnabled   bool
		FoundSecretCount        int
	}

	PushViolationsOutput struct {
		Violations []types.RuleViolations
	}

	PushVerifyOutput struct {
		FileSizeLimit           int64
		PrincipalCommitterMatch bool
		SecretScanningEnabled   bool
		Protections             map[int64]PushProtection
	}

	PushVerifier interface {
		PushVerify(
			ctx context.Context,
			in PushVerifyInput,
		) (PushVerifyOutput, []types.RuleViolations, error)

		Violations(context.Context, *PushViolationsInput) (PushViolationsOutput, error)
	}

	DefPush struct {
		FileSizeLimit           int64 `json:"file_size_limit"`
		PrincipalCommitterMatch bool  `json:"principal_committer_match"`
		SecretScanningEnabled   bool  `json:"secret_scanning_enabled"`
	}
)

func (in *PushViolationsInput) HasViolations() bool {
	return in.FindOversizeFilesOutput != nil && (in.FindOversizeFilesOutput.Total > 0) ||
		in.CommitterMismatchCount > 0 ||
		in.FoundSecretCount > 0
}

func (v *DefPush) PushVerify(
	_ context.Context,
	_ PushVerifyInput,
) (PushVerifyOutput, []types.RuleViolations, error) {
	return PushVerifyOutput{
		FileSizeLimit:           v.FileSizeLimit,
		PrincipalCommitterMatch: v.PrincipalCommitterMatch,
		SecretScanningEnabled:   v.SecretScanningEnabled,
	}, nil, nil
}
