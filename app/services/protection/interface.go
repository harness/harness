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
	"errors"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	// Definition represents a protection rule definition.
	Definition interface {
		// Sanitize validates if the definition is valid and automatically corrects minor issues.
		Sanitize() error

		Protection
	}

	// Protection defines interface for branch protection.
	Protection interface {
		// CanMerge tests if a pull request can be merged.
		CanMerge(ctx context.Context, in CanMergeInput) (CanMergeOutput, []types.RuleViolations, error)
	}

	CanMergeInput struct {
		Actor        *types.Principal
		Membership   *types.Membership
		TargetRepo   *types.Repository
		SourceRepo   *types.Repository
		PullReq      *types.PullReq
		Reviewers    []*types.PullReqReviewer
		Method       enum.MergeMethod
		CheckResults []types.CheckResult
		// TODO: Add code owners
	}

	CanMergeOutput struct {
		DeleteSourceBranch bool
	}
)

var (
	ErrUnrecognizedType    = errors.New("unrecognized protection type")
	ErrAlreadyRegistered   = errors.New("protection type already registered")
	ErrPatternEmpty        = errors.New("pattern doesn't match anything")
	ErrPatternEmptyPattern = errors.New("name pattern can't be empty")
)

func IsCritical(violations []types.RuleViolations) bool {
	for i := range violations {
		if violations[i].IsCritical() {
			return true
		}
	}
	return false
}
