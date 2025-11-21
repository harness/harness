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
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const TypePush enum.RuleType = "push"

// Push implements protection rules for the rule type TypePush.
type Push struct {
	Bypass DefBypass `json:"bypass"`
	Push   DefPush   `json:"push"`
}

var (
	_ Definition     = (*Push)(nil)
	_ PushProtection = (*Push)(nil)
)

func (p *Push) PushVerify(
	ctx context.Context,
	in PushVerifyInput,
) (PushVerifyOutput, []types.RuleViolations, error) {
	out, violations, err := p.Push.PushVerify(ctx, in)
	if err != nil {
		return PushVerifyOutput{}, nil, fmt.Errorf("file size limit verify error: %w", err)
	}

	bypassable := p.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	for i := range violations {
		violations[i].Bypassable = bypassable
		violations[i].Bypassed = bypassable
	}

	return out, violations, nil
}

func (p *Push) Violations(
	ctx context.Context,
	in *PushViolationsInput,
) (PushViolationsOutput, error) {
	var violations types.RuleViolations

	if in.FindOversizeFilesOutput != nil {
		for _, fileInfos := range in.FindOversizeFilesOutput.FileInfos {
			if p.Push.FileSizeLimit > 0 && fileInfos.Size > p.Push.FileSizeLimit {
				violations.Addf(codePushFileSizeLimit,
					"Found file(s) exceeding the filesize limit of %d.",
					p.Push.FileSizeLimit,
				)
				break
			}
		}
	}

	if p.Push.PrincipalCommitterMatch && in.PrincipalCommitterMatch &&
		in.CommitterMismatchCount > 0 {
		violations.Addf(codePushPrincipalCommitterMatch,
			"Committer verification failed for total of %d commit(s).",
			in.CommitterMismatchCount,
		)
	}

	if p.Push.SecretScanningEnabled && in.SecretScanningEnabled &&
		in.FoundSecretCount > 0 {
		violations.Addf(codeSecretScanningEnabled,
			"Found total of %d new secret(s)",
			in.FoundSecretCount,
		)
	}

	bypassable := p.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	violations.Bypassable = bypassable
	violations.Bypassed = bypassable

	return PushViolationsOutput{
		Violations: []types.RuleViolations{violations},
	}, nil
}

func (p *Push) UserIDs() ([]int64, error) {
	return p.Bypass.UserIDs, nil
}

func (p *Push) UserGroupIDs() ([]int64, error) {
	return p.Bypass.UserGroupIDs, nil
}

func (p *Push) Sanitize() error {
	if err := p.Bypass.Sanitize(); err != nil {
		return fmt.Errorf("bypass: %w", err)
	}

	return nil
}
