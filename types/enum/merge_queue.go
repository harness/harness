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

// MergeQueueEntryState defines the state of a pull request entry in the merge queue.
type MergeQueueEntryState string

func (MergeQueueEntryState) Enum() []any { return toInterfaceSlice(mergeQueueEntryStates) }
func (s MergeQueueEntryState) Sanitize() (MergeQueueEntryState, bool) {
	return Sanitize(s, GetAllMergeQueueEntryStates)
}
func GetAllMergeQueueEntryStates() ([]MergeQueueEntryState, MergeQueueEntryState) {
	return mergeQueueEntryStates, ""
}

// MergeQueueEntryState enumeration.
const (
	// MergeQueueEntryStateMergePending indicates the entry's merge commit is being created.
	MergeQueueEntryStateMergePending MergeQueueEntryState = "merge_pending"
	// MergeQueueEntryStateChecksPending indicates the merge queue checks have not yet been started.
	MergeQueueEntryStateChecksPending MergeQueueEntryState = "checks_pending"
	// MergeQueueEntryStateChecksInProgress indicates the merge commit has been created
	// and CI checks are running against it.
	MergeQueueEntryStateChecksInProgress MergeQueueEntryState = "checks_in_progress"
	// MergeQueueEntryStateMergeGroup indicates the entry is part of the active merge group
	// and is waiting to be fast-forwarded onto the target branch.
	MergeQueueEntryStateMergeGroup MergeQueueEntryState = "merge_group"
)

var mergeQueueEntryStates = sortEnum([]MergeQueueEntryState{
	MergeQueueEntryStateMergePending,
	MergeQueueEntryStateChecksPending,
	MergeQueueEntryStateChecksInProgress,
	MergeQueueEntryStateMergeGroup,
})

// MergeQueueRemovalReason defines the reason a pull request was removed from the merge queue.
type MergeQueueRemovalReason string

func (MergeQueueRemovalReason) Enum() []any { return toInterfaceSlice(mergeQueueRemovalReasons) }
func (r MergeQueueRemovalReason) Sanitize() (MergeQueueRemovalReason, bool) {
	return Sanitize(r, GetAllMergeQueueRemovalReasons)
}
func GetAllMergeQueueRemovalReasons() ([]MergeQueueRemovalReason, MergeQueueRemovalReason) {
	return mergeQueueRemovalReasons, ""
}

// MergeQueueRemovalReason enumeration.
const (
	// MergeQueueRemovalReasonManual indicates the entry was deliberately removed by a user.
	MergeQueueRemovalReasonManual MergeQueueRemovalReason = "manual"
	// MergeQueueRemovalReasonConflict indicates the entry was removed due to a merge conflict.
	MergeQueueRemovalReasonConflict MergeQueueRemovalReason = "conflict"
	// MergeQueueRemovalReasonCheckFail indicates the entry was removed because CI checks failed.
	MergeQueueRemovalReasonCheckFail MergeQueueRemovalReason = "check_fail"
	// MergeQueueRemovalReasonNotQueueable indicates the entry was removed because pull request couldn't be queued.
	MergeQueueRemovalReasonNotQueueable MergeQueueRemovalReason = "not_queueable"
	// MergeQueueRemovalReasonNoQueue indicates the entry was removed because merge queue isn't configured.
	MergeQueueRemovalReasonNoQueue MergeQueueRemovalReason = "no_queue"
	// MergeQueueRemovalReasonError indicates the entry was removed due to an unexpected error.
	MergeQueueRemovalReasonError MergeQueueRemovalReason = "error"
)

var mergeQueueRemovalReasons = sortEnum([]MergeQueueRemovalReason{
	MergeQueueRemovalReasonManual,
	MergeQueueRemovalReasonConflict,
	MergeQueueRemovalReasonCheckFail,
	MergeQueueRemovalReasonNotQueueable,
	MergeQueueRemovalReasonNoQueue,
	MergeQueueRemovalReasonError,
})
