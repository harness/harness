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

package mergequeue

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
)

var errMergeConflict = errors.New("merge conflict")

func (s *Service) createMergeCommit(
	ctx context.Context,
	writeParams git.WriteParams,
	baseCommitSHA sha.SHA,
	headCommitSHA sha.SHA,
	entry *types.MergeQueueEntry,
	input merge.PullReqGitInput,
) (git.MergeOutput, error) {
	now := time.Now().UTC()

	mergeOutput, err := s.git.Merge(ctx, &git.MergeParams{
		WriteParams:      writeParams,
		BaseSHA:          baseCommitSHA,
		HeadSHA:          headCommitSHA,
		Message:          input.CommitMessage,
		Committer:        input.Committer,
		CommitterDate:    &now,
		Author:           input.Author,
		AuthorDate:       &now,
		Refs:             nil, // Update no refs!
		Force:            false,
		DeleteHeadBranch: false,
		Method:           gitenum.MergeMethod(entry.MergeMethod),
	})
	if err != nil {
		return git.MergeOutput{}, fmt.Errorf("failed to create merge queue commit: %w", err)
	}

	if mergeOutput.MergeSHA.IsEmpty() || len(mergeOutput.ConflictFiles) > 0 {
		return git.MergeOutput{}, errMergeConflict
	}

	return mergeOutput, nil
}
