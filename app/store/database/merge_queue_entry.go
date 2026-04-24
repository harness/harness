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

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.MergeQueueEntryStore = (*MergeQueueEntryStore)(nil)

func NewMergeQueueEntryStore(db *sqlx.DB) *MergeQueueEntryStore {
	return &MergeQueueEntryStore{db: db}
}

type MergeQueueEntryStore struct {
	db *sqlx.DB
}

type mergeQueueEntry struct {
	PullReqID          int64       `db:"merge_queue_entry_pullreq_id"`
	MergeQueueID       int64       `db:"merge_queue_entry_queue_id"`
	Version            int64       `db:"merge_queue_entry_version"`
	CreatedBy          int64       `db:"merge_queue_entry_created_by"`
	Created            int64       `db:"merge_queue_entry_created"`
	Updated            int64       `db:"merge_queue_entry_updated"`
	OrderIndex         int64       `db:"merge_queue_entry_order_index"`
	State              string      `db:"merge_queue_entry_state"`
	BaseCommitSHA      null.String `db:"merge_queue_entry_base_commit_sha"`
	HeadCommitSHA      null.String `db:"merge_queue_entry_head_commit_sha"`
	MergeCommitSHA     null.String `db:"merge_queue_entry_merge_commit_sha"`
	MergeBaseSHA       null.String `db:"merge_queue_entry_merge_base_sha"`
	CommitCount        int         `db:"merge_queue_entry_commit_count"`
	ChangedFileCount   int         `db:"merge_queue_entry_changed_file_count"`
	Additions          int         `db:"merge_queue_entry_additions"`
	Deletions          int         `db:"merge_queue_entry_deletions"`
	ChecksCommitSHA    null.String `db:"merge_queue_entry_checks_commit_sha"`
	ChecksStarted      null.Int    `db:"merge_queue_entry_checks_started"`
	ChecksDeadline     null.Int    `db:"merge_queue_entry_checks_deadline"`
	MergeMethod        string      `db:"merge_queue_entry_merge_method"`
	CommitTitle        string      `db:"merge_queue_entry_commit_title"`
	CommitMessage      string      `db:"merge_queue_entry_commit_message"`
	DeleteSourceBranch bool        `db:"merge_queue_entry_delete_source_branch"`
}

const (
	mergeQueueEntryColumns = `
		 merge_queue_entry_pullreq_id
		,merge_queue_entry_queue_id
		,merge_queue_entry_version
		,merge_queue_entry_created_by
		,merge_queue_entry_created
		,merge_queue_entry_updated
		,merge_queue_entry_order_index
		,merge_queue_entry_state
		,merge_queue_entry_base_commit_sha
		,merge_queue_entry_head_commit_sha
		,merge_queue_entry_merge_commit_sha
		,merge_queue_entry_merge_base_sha
		,merge_queue_entry_commit_count
		,merge_queue_entry_changed_file_count
		,merge_queue_entry_additions
		,merge_queue_entry_deletions
		,merge_queue_entry_checks_commit_sha
		,merge_queue_entry_checks_started
		,merge_queue_entry_checks_deadline
		,merge_queue_entry_merge_method
		,merge_queue_entry_commit_title
		,merge_queue_entry_commit_message
		,merge_queue_entry_delete_source_branch`

	mergeQueueEntrySelectBase = `
	SELECT` + mergeQueueEntryColumns + `
	FROM merge_queue_entries`
)

func (s *MergeQueueEntryStore) Find(ctx context.Context, pullReqID int64) (*types.MergeQueueEntry, error) {
	const sqlQuery = mergeQueueEntrySelectBase + `
	WHERE merge_queue_entry_pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst mergeQueueEntry
	if err := db.GetContext(ctx, &dst, sqlQuery, pullReqID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find merge queue entry")
	}

	result, err := s.mapExternal(&dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map merge queue entry: %w", err)
	}

	return result, nil
}

func (s *MergeQueueEntryStore) FindByMergeCommit(
	ctx context.Context,
	mergeCommitSHA sha.SHA,
) (*types.MergeQueueEntry, error) {
	const sqlQuery = mergeQueueEntrySelectBase + `
	WHERE merge_queue_entry_merge_commit_sha = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst mergeQueueEntry
	if err := db.GetContext(ctx, &dst, sqlQuery, mergeCommitSHA.String()); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find merge queue entry by merge commit")
	}

	result, err := s.mapExternal(&dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map merge queue entry: %w", err)
	}

	return result, nil
}

func (s *MergeQueueEntryStore) Create(ctx context.Context, e *types.MergeQueueEntry) error {
	const sqlQuery = `
	INSERT INTO merge_queue_entries (
		 merge_queue_entry_pullreq_id
		,merge_queue_entry_queue_id
		,merge_queue_entry_version
		,merge_queue_entry_created_by
		,merge_queue_entry_created
		,merge_queue_entry_updated
		,merge_queue_entry_order_index
		,merge_queue_entry_state
		,merge_queue_entry_base_commit_sha
		,merge_queue_entry_head_commit_sha
		,merge_queue_entry_merge_commit_sha
		,merge_queue_entry_merge_base_sha
		,merge_queue_entry_commit_count
		,merge_queue_entry_changed_file_count
		,merge_queue_entry_additions
		,merge_queue_entry_deletions
		,merge_queue_entry_checks_commit_sha
		,merge_queue_entry_checks_started
		,merge_queue_entry_checks_deadline
		,merge_queue_entry_merge_method
		,merge_queue_entry_commit_title
		,merge_queue_entry_commit_message
		,merge_queue_entry_delete_source_branch
	) values (
		 :merge_queue_entry_pullreq_id
		,:merge_queue_entry_queue_id
		,:merge_queue_entry_version
		,:merge_queue_entry_created_by
		,:merge_queue_entry_created
		,:merge_queue_entry_updated
		,:merge_queue_entry_order_index
		,:merge_queue_entry_state
		,:merge_queue_entry_base_commit_sha
		,:merge_queue_entry_head_commit_sha
		,:merge_queue_entry_merge_commit_sha
		,:merge_queue_entry_merge_base_sha
		,:merge_queue_entry_commit_count
		,:merge_queue_entry_changed_file_count
		,:merge_queue_entry_additions
		,:merge_queue_entry_deletions
		,:merge_queue_entry_checks_commit_sha
		,:merge_queue_entry_checks_started
		,:merge_queue_entry_checks_deadline
		,:merge_queue_entry_merge_method
		,:merge_queue_entry_commit_title
		,:merge_queue_entry_commit_message
		,:merge_queue_entry_delete_source_branch
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	dbEntry := s.mapInternal(e)

	query, arg, err := db.BindNamed(sqlQuery, dbEntry)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind merge queue entry object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert merge queue entry")
	}

	return nil
}

func (s *MergeQueueEntryStore) Update(ctx context.Context, e *types.MergeQueueEntry) error {
	const sqlQuery = `
	UPDATE merge_queue_entries
	SET
		 merge_queue_entry_version = :merge_queue_entry_version
		,merge_queue_entry_updated = :merge_queue_entry_updated
		,merge_queue_entry_order_index = :merge_queue_entry_order_index
		,merge_queue_entry_state = :merge_queue_entry_state
		,merge_queue_entry_base_commit_sha = :merge_queue_entry_base_commit_sha
		,merge_queue_entry_head_commit_sha = :merge_queue_entry_head_commit_sha
		,merge_queue_entry_merge_commit_sha = :merge_queue_entry_merge_commit_sha
		,merge_queue_entry_merge_base_sha = :merge_queue_entry_merge_base_sha
		,merge_queue_entry_commit_count = :merge_queue_entry_commit_count
		,merge_queue_entry_changed_file_count = :merge_queue_entry_changed_file_count
		,merge_queue_entry_additions = :merge_queue_entry_additions
		,merge_queue_entry_deletions = :merge_queue_entry_deletions
		,merge_queue_entry_checks_commit_sha = :merge_queue_entry_checks_commit_sha
		,merge_queue_entry_checks_started = :merge_queue_entry_checks_started
		,merge_queue_entry_checks_deadline = :merge_queue_entry_checks_deadline
		,merge_queue_entry_merge_method = :merge_queue_entry_merge_method
		,merge_queue_entry_commit_title = :merge_queue_entry_commit_title
		,merge_queue_entry_commit_message = :merge_queue_entry_commit_message
		,merge_queue_entry_delete_source_branch = :merge_queue_entry_delete_source_branch
	WHERE merge_queue_entry_pullreq_id = :merge_queue_entry_pullreq_id
	  AND merge_queue_entry_version = :merge_queue_entry_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbEntry := s.mapInternal(e)

	dbEntry.Version++
	dbEntry.Updated = updatedAt.UnixMilli()

	updatedMergeQueue, err := s.mapExternal(dbEntry)
	if err != nil {
		return fmt.Errorf("failed to map merge queue entry: %w", err)
	}

	query, arg, err := db.BindNamed(sqlQuery, dbEntry)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind merge queue entry object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update merge queue entry")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	*e = *updatedMergeQueue

	return nil
}

func (s *MergeQueueEntryStore) UpdateOptLock(ctx context.Context,
	e *types.MergeQueueEntry,
	mutateFn func(e *types.MergeQueueEntry) error,
) (*types.MergeQueueEntry, error) {
	for {
		dup := *e

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		e, err = s.Find(ctx, e.PullReqID)
		if err != nil {
			return nil, err
		}
	}
}

func (s *MergeQueueEntryStore) Delete(ctx context.Context, pullReqID int64) error {
	const sqlQuery = `
	DELETE FROM merge_queue_entries
	WHERE merge_queue_entry_pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, pullReqID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete merge queue entry")
	}

	return nil
}

func (s *MergeQueueEntryStore) DeleteAllForMergeQueue(ctx context.Context, mergeQueueID int64) error {
	const sqlQuery = `
	DELETE FROM merge_queue_entries
	WHERE merge_queue_entry_queue_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, mergeQueueID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete all merge queue entries")
	}

	return nil
}

func (s *MergeQueueEntryStore) ListForMergeQueue(
	ctx context.Context,
	mergeQueueID int64,
) ([]*types.MergeQueueEntry, error) {
	const sqlQuery = mergeQueueEntrySelectBase + `
	WHERE merge_queue_entry_queue_id = $1
	ORDER BY merge_queue_entry_order_index ASC`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]*mergeQueueEntry, 0)
	if err := db.SelectContext(ctx, &dst, sqlQuery, mergeQueueID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list merge queue entries")
	}

	result, err := s.mapExternalList(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map merge queue entries: %w", err)
	}

	return result, nil
}

func (s *MergeQueueEntryStore) CountForRepoAndBranch(
	ctx context.Context,
	repoID int64,
	branch string,
) (int64, error) {
	const sqlQuery = `
	SELECT COUNT(*)
	FROM merge_queue_entries
	JOIN merge_queues ON merge_queue_entries.merge_queue_entry_queue_id = merge_queues.merge_queue_id
	WHERE merge_queues.merge_queue_repo_id = $1
	  AND merge_queues.merge_queue_branch = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err := db.QueryRowContext(ctx, sqlQuery, repoID, branch).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count merge queue entries")
	}

	return count, nil
}

func (s *MergeQueueEntryStore) ListOverdueChecks(
	ctx context.Context,
	now int64,
) ([]*types.MergeQueueEntry, error) {
	const sqlQuery = mergeQueueEntrySelectBase + `
	WHERE merge_queue_entry_state = 'checks_in_progress'
	  AND merge_queue_entry_checks_deadline IS NOT NULL
	  AND merge_queue_entry_checks_deadline < $1
	ORDER BY merge_queue_entry_checks_deadline ASC`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]*mergeQueueEntry, 0)
	if err := db.SelectContext(ctx, &dst, sqlQuery, now); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list overdue merge queue entries")
	}

	result, err := s.mapExternalList(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map merge queue entries: %w", err)
	}

	return result, nil
}

func (*MergeQueueEntryStore) mapExternal(e *mergeQueueEntry) (*types.MergeQueueEntry, error) {
	baseCommitSHA, err := sha.NewOrEmpty(e.BaseCommitSHA.String)
	if err != nil {
		return nil, fmt.Errorf("failed to convert base commit SHA: %w", err)
	}
	headCommitSHA, err := sha.NewOrEmpty(e.HeadCommitSHA.String)
	if err != nil {
		return nil, fmt.Errorf("failed to convert head commit SHA: %w", err)
	}
	mergeCommitSHA, err := sha.NewOrEmpty(e.MergeCommitSHA.String)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merge commit SHA: %w", err)
	}
	mergeBaseSHA, err := sha.NewOrEmpty(e.MergeBaseSHA.String)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merge base SHA: %w", err)
	}
	checksCommitSHA, err := sha.NewOrEmpty(e.ChecksCommitSHA.String)
	if err != nil {
		return nil, fmt.Errorf("failed to convert checks commit SHA: %w", err)
	}

	state := enum.MergeQueueEntryState(e.State)
	if _, ok := state.Sanitize(); !ok {
		return nil, fmt.Errorf("invalid state: %s", e.State)
	}

	return &types.MergeQueueEntry{
		PullReqID:          e.PullReqID,
		MergeQueueID:       e.MergeQueueID,
		Version:            e.Version,
		CreatedBy:          e.CreatedBy,
		Created:            e.Created,
		Updated:            e.Updated,
		OrderIndex:         e.OrderIndex,
		State:              state,
		BaseCommitSHA:      baseCommitSHA,
		HeadCommitSHA:      headCommitSHA,
		MergeCommitSHA:     mergeCommitSHA,
		MergeBaseSHA:       mergeBaseSHA,
		CommitCount:        e.CommitCount,
		ChangedFileCount:   e.ChangedFileCount,
		Additions:          e.Additions,
		Deletions:          e.Deletions,
		ChecksCommitSHA:    checksCommitSHA,
		ChecksStarted:      e.ChecksStarted.Ptr(),
		ChecksDeadline:     e.ChecksDeadline.Ptr(),
		MergeMethod:        enum.MergeMethod(e.MergeMethod),
		CommitTitle:        e.CommitTitle,
		CommitMessage:      e.CommitMessage,
		DeleteSourceBranch: e.DeleteSourceBranch,
	}, nil
}

func (s *MergeQueueEntryStore) mapExternalList(e []*mergeQueueEntry) (list []*types.MergeQueueEntry, err error) {
	list = make([]*types.MergeQueueEntry, len(e))
	for i, qe := range e {
		list[i], err = s.mapExternal(qe)
		if err != nil {
			return nil, fmt.Errorf("failed to map merge queue entry: %w", err)
		}
	}
	return list, nil
}

func (*MergeQueueEntryStore) mapInternal(e *types.MergeQueueEntry) *mergeQueueEntry {
	baseCommitSHA := null.NewString(e.BaseCommitSHA.String(), !e.BaseCommitSHA.IsEmpty())
	headCommitSHA := null.NewString(e.HeadCommitSHA.String(), !e.HeadCommitSHA.IsEmpty())
	mergeCommitSHA := null.NewString(e.MergeCommitSHA.String(), !e.MergeCommitSHA.IsEmpty())
	mergeBaseSHA := null.NewString(e.MergeBaseSHA.String(), !e.MergeBaseSHA.IsEmpty())
	checksCommitSHA := null.NewString(e.ChecksCommitSHA.String(), !e.ChecksCommitSHA.IsEmpty())

	return &mergeQueueEntry{
		PullReqID:          e.PullReqID,
		MergeQueueID:       e.MergeQueueID,
		Version:            e.Version,
		CreatedBy:          e.CreatedBy,
		Created:            e.Created,
		Updated:            e.Updated,
		OrderIndex:         e.OrderIndex,
		State:              string(e.State),
		BaseCommitSHA:      baseCommitSHA,
		HeadCommitSHA:      headCommitSHA,
		MergeCommitSHA:     mergeCommitSHA,
		MergeBaseSHA:       mergeBaseSHA,
		CommitCount:        e.CommitCount,
		ChangedFileCount:   e.ChangedFileCount,
		Additions:          e.Additions,
		Deletions:          e.Deletions,
		ChecksCommitSHA:    checksCommitSHA,
		ChecksStarted:      null.IntFromPtr(e.ChecksStarted),
		ChecksDeadline:     null.IntFromPtr(e.ChecksDeadline),
		MergeMethod:        string(e.MergeMethod),
		CommitTitle:        e.CommitTitle,
		CommitMessage:      e.CommitMessage,
		DeleteSourceBranch: e.DeleteSourceBranch,
	}
}
