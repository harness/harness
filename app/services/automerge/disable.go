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

package automerge

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var (
	ErrPullReqNotOpen = errors.New("pull request is not open")
	ErrPullReqDraft   = errors.New("pull request must not be draft")
	ErrNotInAutoMerge = errors.New("not in auto merge mode")
)

// Disable disables auto-merge on a pull request. The PR must be open
// and in the auto-merge substate, otherwise an error is returned.
func (s *Service) Disable(
	ctx context.Context,
	targetRepo *types.RepositoryCore,
	prNumber int64,
) (*types.PullReq, error) {
	var pr *types.PullReq

	err := controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		var err error

		pr, err = s.pullreqStore.FindByNumber(ctx, targetRepo.ID, prNumber)
		if err != nil {
			return fmt.Errorf("failed to get pull request by number: %w", err)
		}

		if pr.State != enum.PullReqStateOpen {
			return ErrPullReqNotOpen
		}

		if pr.IsDraft {
			return ErrPullReqDraft
		}

		if pr.SubState != enum.PullReqSubStateAutoMerge {
			return ErrNotInAutoMerge
		}

		pr.SubState = enum.PullReqSubStateNone

		err = s.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		_, err = s.autoMergeStore.Delete(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to update auto merge state: %w", err)
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return nil, fmt.Errorf("failed to disable auto merge for the pull request: %w", err)
	}

	return pr, nil
}
