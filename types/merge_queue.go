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

package types

import (
	"fmt"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"
)

type MergeQueue struct {
	ID            int64
	RepoID        int64
	Branch        string
	Version       int64
	Created       int64
	Updated       int64
	OrderSequence int64
}

type MergeQueueEntry struct {
	PullReqID    int64
	MergeQueueID int64

	Version   int64
	CreatedBy int64
	Created   int64
	Updated   int64

	// Index in the merge queue.
	OrderIndex int64

	// State of the merge queue entry.
	State enum.MergeQueueEntryState

	// BaseCommitSHA is the commit on which the changes should be applied.
	// Should be either the last commit on the PR's target branch or
	// the MergeCommitSHA of the merge queue's previous entry.
	BaseCommitSHA sha.SHA

	// HeadCommitSHA is the SourceCommitSHA from the PR - the latest commit on the PR's source branch.
	HeadCommitSHA sha.SHA

	// MergeCommitSHA is the merge commit created by the merge queue.
	MergeCommitSHA sha.SHA

	// MergeBaseSHA is the base commit SHA used when creating the merge commit.
	MergeBaseSHA sha.SHA

	// CommitCount is the number of commits included in the merge.
	CommitCount int

	// ChangedFileCount is the number of files changed in the merge.
	ChangedFileCount int

	// Additions is the total number of added lines.
	Additions int

	// Deletions is the total number of deleted lines.
	Deletions int

	// ChecksCommitSHA is the MergeCommitSHA of the chain leader entry
	// against which the merge queue checks are run.
	ChecksCommitSHA sha.SHA

	// ChecksStarted is the timestamp when the checks have been started.
	ChecksStarted *int64

	// ChecksDeadline is the timestamp by which the checks must complete.
	ChecksDeadline *int64

	// MergeMethod is the merge method to use when merging the PR.
	MergeMethod enum.MergeMethod

	// CommitTitle is the title of the merge commit (used as the commit subject line).
	CommitTitle string

	// CommitMessage is the body of the merge commit message.
	CommitMessage string

	// DeleteSourceBranch indicates whether the source branch should be deleted after merging.
	DeleteSourceBranch bool
}

func (entry *MergeQueueEntry) String() string {
	return fmt.Sprintf("id=%d q=%d index=%d method=%s state=%s merge=%s checks=%s",
		entry.PullReqID, entry.MergeQueueID, entry.OrderIndex, entry.MergeMethod, entry.State,
		entry.MergeCommitSHA, entry.ChecksCommitSHA)
}

func (entry *MergeQueueEntry) CancelMerge() {
	entry.State = enum.MergeQueueEntryStateMergePending
	entry.MergeCommitSHA = sha.None
	entry.CommitCount = 0
	entry.ChangedFileCount = 0
	entry.Additions = 0
	entry.Deletions = 0
	entry.ChecksCommitSHA = sha.None
	entry.ChecksStarted = nil
	entry.ChecksDeadline = nil
}
