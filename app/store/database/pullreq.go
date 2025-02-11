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
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

var _ store.PullReqStore = (*PullReqStore)(nil)

// NewPullReqStore returns a new PullReqStore.
func NewPullReqStore(db *sqlx.DB,
	pCache store.PrincipalInfoCache) *PullReqStore {
	return &PullReqStore{
		db:     db,
		pCache: pCache,
	}
}

// PullReqStore implements store.PullReqStore backed by a relational database.
type PullReqStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

// pullReq is used to fetch pull request data from the database.
// The object should be later re-packed into a different struct to return it as an API response.
type pullReq struct {
	ID      int64 `db:"pullreq_id"`
	Version int64 `db:"pullreq_version"`
	Number  int64 `db:"pullreq_number"`

	CreatedBy int64    `db:"pullreq_created_by"`
	Created   int64    `db:"pullreq_created"`
	Updated   int64    `db:"pullreq_updated"`
	Edited    int64    `db:"pullreq_edited"` // TODO: Remove
	Closed    null.Int `db:"pullreq_closed"`

	State   enum.PullReqState `db:"pullreq_state"`
	IsDraft bool              `db:"pullreq_is_draft"`

	CommentCount    int `db:"pullreq_comment_count"`
	UnresolvedCount int `db:"pullreq_unresolved_count"`

	Title       string `db:"pullreq_title"`
	Description string `db:"pullreq_description"`

	SourceRepoID int64  `db:"pullreq_source_repo_id"`
	SourceBranch string `db:"pullreq_source_branch"`
	SourceSHA    string `db:"pullreq_source_sha"`
	TargetRepoID int64  `db:"pullreq_target_repo_id"`
	TargetBranch string `db:"pullreq_target_branch"`

	ActivitySeq int64 `db:"pullreq_activity_seq"`

	MergedBy    null.Int    `db:"pullreq_merged_by"`
	Merged      null.Int    `db:"pullreq_merged"`
	MergeMethod null.String `db:"pullreq_merge_method"`

	MergeTargetSHA null.String `db:"pullreq_merge_target_sha"`
	MergeBaseSHA   string      `db:"pullreq_merge_base_sha"`
	MergeSHA       null.String `db:"pullreq_merge_sha"`

	MergeCheckStatus  enum.MergeCheckStatus `db:"pullreq_merge_check_status"`
	MergeConflicts    null.String           `db:"pullreq_merge_conflicts"`
	RebaseCheckStatus enum.MergeCheckStatus `db:"pullreq_rebase_check_status"`
	RebaseConflicts   null.String           `db:"pullreq_rebase_conflicts,omitempty"`

	CommitCount null.Int `db:"pullreq_commit_count"`
	FileCount   null.Int `db:"pullreq_file_count"`
	Additions   null.Int `db:"pullreq_additions"`
	Deletions   null.Int `db:"pullreq_deletions"`
}

const (
	pullReqColumnsNoDescription = `
		 pullreq_id
		,pullreq_version
		,pullreq_number
		,pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_edited
		,pullreq_closed
		,pullreq_state
		,pullreq_is_draft
		,pullreq_comment_count
		,pullreq_unresolved_count
		,pullreq_title
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_source_sha
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_activity_seq
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_method
		,pullreq_merge_target_sha
		,pullreq_merge_base_sha
		,pullreq_merge_sha
		,pullreq_merge_check_status
		,pullreq_merge_conflicts
		,pullreq_rebase_check_status
		,pullreq_rebase_conflicts
		,pullreq_commit_count
		,pullreq_file_count
		,pullreq_additions
		,pullreq_deletions`

	pullReqColumns = pullReqColumnsNoDescription + `
		,pullreq_description`

	pullReqSelectBase = `
	SELECT` + pullReqColumns + `
	FROM pullreqs`
)

// Find finds the pull request by id.
func (s *PullReqStore) Find(ctx context.Context, id int64) (*types.PullReq, error) {
	const sqlQuery = pullReqSelectBase + `
	WHERE pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReq{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request")
	}

	return s.mapPullReq(ctx, dst), nil
}

func (s *PullReqStore) findByNumberInternal(
	ctx context.Context,
	repoID,
	number int64,
	lock bool,
) (*types.PullReq, error) {
	sqlQuery := pullReqSelectBase + `
	WHERE pullreq_target_repo_id = $1 AND pullreq_number = $2`

	if lock && !strings.HasPrefix(s.db.DriverName(), "sqlite") {
		sqlQuery += "\n" + database.SQLForUpdate
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReq{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, number); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request by number")
	}

	return s.mapPullReq(ctx, dst), nil
}

// FindByNumberWithLock finds the pull request by repo ID and pull request number
// and locks the pull request for the duration of the transaction.
func (s *PullReqStore) FindByNumberWithLock(
	ctx context.Context,
	repoID,
	number int64,
) (*types.PullReq, error) {
	return s.findByNumberInternal(ctx, repoID, number, true)
}

// FindByNumber finds the pull request by repo ID and pull request number.
func (s *PullReqStore) FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReq, error) {
	return s.findByNumberInternal(ctx, repoID, number, false)
}

// Create creates a new pull request.
func (s *PullReqStore) Create(ctx context.Context, pr *types.PullReq) error {
	const sqlQuery = `
	INSERT INTO pullreqs (
		 pullreq_version
		,pullreq_number
		,pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_edited
		,pullreq_closed
		,pullreq_state
		,pullreq_is_draft
		,pullreq_comment_count
		,pullreq_unresolved_count
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_source_sha
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_activity_seq
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_method
		,pullreq_merge_target_sha
		,pullreq_merge_base_sha
		,pullreq_merge_sha
		,pullreq_merge_check_status
		,pullreq_merge_conflicts
		,pullreq_rebase_check_status
		,pullreq_rebase_conflicts
		,pullreq_commit_count
		,pullreq_file_count
		,pullreq_additions
		,pullreq_deletions
	) values (
		 :pullreq_version
		,:pullreq_number
		,:pullreq_created_by
		,:pullreq_created
		,:pullreq_updated
		,:pullreq_edited
		,:pullreq_closed
		,:pullreq_state
		,:pullreq_is_draft
		,:pullreq_comment_count
		,:pullreq_unresolved_count
		,:pullreq_title
		,:pullreq_description
		,:pullreq_source_repo_id
		,:pullreq_source_branch
		,:pullreq_source_sha
		,:pullreq_target_repo_id
		,:pullreq_target_branch
		,:pullreq_activity_seq
		,:pullreq_merged_by
		,:pullreq_merged
		,:pullreq_merge_method
		,:pullreq_merge_target_sha
		,:pullreq_merge_base_sha
		,:pullreq_merge_sha
		,:pullreq_merge_check_status
		,:pullreq_merge_conflicts
		,:pullreq_rebase_check_status
		,:pullreq_rebase_conflicts
		,:pullreq_commit_count
		,:pullreq_file_count
		,:pullreq_additions
		,:pullreq_deletions
	) RETURNING pullreq_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReq(pr))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pullReq object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&pr.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// Update updates the pull request.
func (s *PullReqStore) Update(ctx context.Context, pr *types.PullReq) error {
	const sqlQuery = `
	UPDATE pullreqs
	SET
	     pullreq_version = :pullreq_version
		,pullreq_updated = :pullreq_updated
		,pullreq_edited = :pullreq_edited
		,pullreq_closed = :pullreq_closed
		,pullreq_state = :pullreq_state
		,pullreq_is_draft = :pullreq_is_draft
		,pullreq_comment_count = :pullreq_comment_count
		,pullreq_unresolved_count = :pullreq_unresolved_count
		,pullreq_title = :pullreq_title
		,pullreq_description = :pullreq_description
		,pullreq_activity_seq = :pullreq_activity_seq
		,pullreq_source_sha = :pullreq_source_sha
		,pullreq_merged_by = :pullreq_merged_by
		,pullreq_merged = :pullreq_merged
		,pullreq_merge_method = :pullreq_merge_method
		,pullreq_merge_target_sha = :pullreq_merge_target_sha
		,pullreq_merge_base_sha = :pullreq_merge_base_sha
		,pullreq_merge_sha = :pullreq_merge_sha
		,pullreq_merge_check_status = :pullreq_merge_check_status
		,pullreq_merge_conflicts = :pullreq_merge_conflicts
		,pullreq_rebase_check_status = :pullreq_rebase_check_status
		,pullreq_rebase_conflicts = :pullreq_rebase_conflicts
		,pullreq_commit_count = :pullreq_commit_count
		,pullreq_file_count = :pullreq_file_count
		,pullreq_additions = :pullreq_additions
		,pullreq_deletions = :pullreq_deletions
	WHERE pullreq_id = :pullreq_id AND pullreq_version = :pullreq_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now().UnixMilli()

	dbPR := mapInternalPullReq(pr)
	dbPR.Version++
	dbPR.Updated = updatedAt
	dbPR.Edited = updatedAt

	query, arg, err := db.BindNamed(sqlQuery, dbPR)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update pull request")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	*pr = *s.mapPullReq(ctx, dbPR)

	return nil
}

// UpdateOptLock the pull request details using the optimistic locking mechanism.
func (s *PullReqStore) UpdateOptLock(ctx context.Context, pr *types.PullReq,
	mutateFn func(pr *types.PullReq) error,
) (*types.PullReq, error) {
	for {
		dup := *pr

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

		pr, err = s.Find(ctx, pr.ID)
		if err != nil {
			return nil, err
		}
	}
}

// UpdateActivitySeq updates the pull request's activity sequence.
func (s *PullReqStore) UpdateActivitySeq(ctx context.Context, pr *types.PullReq) (*types.PullReq, error) {
	return s.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.ActivitySeq++
		return nil
	})
}

// ResetMergeCheckStatus resets the pull request's mergeability status to unchecked
// for all pr which target branch points to targetBranch.
func (s *PullReqStore) ResetMergeCheckStatus(
	ctx context.Context,
	targetRepo int64,
	targetBranch string,
) error {
	// NOTE: keep pullreq_merge_base_sha on old value as it's a required field.
	const query = `
	UPDATE pullreqs
	SET
		 pullreq_updated = $1
		,pullreq_version = pullreq_version + 1
		,pullreq_merge_target_sha = NULL
		,pullreq_merge_sha = NULL
		,pullreq_merge_check_status = $2
		,pullreq_merge_conflicts = NULL
		,pullreq_rebase_check_status = $2
		,pullreq_rebase_conflicts = NULL
		,pullreq_commit_count = NULL
		,pullreq_file_count = NULL
		,pullreq_additions = NULL
		,pullreq_deletions = NULL
	WHERE pullreq_target_repo_id = $3 AND
		pullreq_target_branch = $4 AND
		pullreq_state not in ($5, $6)`

	db := dbtx.GetAccessor(ctx, s.db)

	now := time.Now().UnixMilli()

	_, err := db.ExecContext(ctx, query, now, enum.MergeCheckStatusUnchecked, targetRepo, targetBranch,
		enum.PullReqStateClosed, enum.PullReqStateMerged)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to reset mergeable status check in pull requests")
	}

	return nil
}

// Delete the pull request.
func (s *PullReqStore) Delete(ctx context.Context, id int64) error {
	const pullReqDelete = `DELETE FROM pullreqs WHERE pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, pullReqDelete, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

// Count of pull requests for a repo.
func (s *PullReqStore) Count(ctx context.Context, opts *types.PullReqFilter) (int64, error) {
	var stmt squirrel.SelectBuilder

	if len(opts.LabelID) > 0 || len(opts.ValueID) > 0 {
		stmt = database.Builder.Select("1")
	} else {
		stmt = database.Builder.Select("COUNT(*)")
	}

	stmt = stmt.From("pullreqs")
	s.applyFilter(&stmt, opts)

	if len(opts.LabelID) > 0 || len(opts.ValueID) > 0 {
		stmt = database.Builder.Select("COUNT(*)").FromSelect(stmt, "subquery")
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of pull requests for a repo.
func (s *PullReqStore) List(ctx context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
	stmt := s.listQuery(opts)

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	// NOTE: string concatenation is safe because the
	// order attribute is an enum and is not user-defined,
	// and is therefore not subject to injection attacks.
	opts.Sort, _ = opts.Sort.Sanitize()
	stmt = stmt.OrderBy("pullreq_" + string(opts.Sort) + " " + opts.Order.String())

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	dst := make([]*pullReq, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	result, err := s.mapSlicePullReq(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Stream returns a list of pull requests for a repo.
func (s *PullReqStore) Stream(ctx context.Context, opts *types.PullReqFilter) (<-chan *types.PullReq, <-chan error) {
	stmt := s.listQuery(opts)

	stmt = stmt.OrderBy("pullreq_updated desc")

	chPRs := make(chan *types.PullReq)
	chErr := make(chan error, 1)

	go func() {
		defer close(chPRs)
		defer close(chErr)

		sql, args, err := stmt.ToSql()
		if err != nil {
			chErr <- fmt.Errorf("failed to convert query to sql: %w", err)
			return
		}

		db := dbtx.GetAccessor(ctx, s.db)

		rows, err := db.QueryxContext(ctx, sql, args...)
		if err != nil {
			chErr <- database.ProcessSQLErrorf(ctx, err, "Failed to execute stream query")
			return
		}

		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var prData pullReq
			err = rows.StructScan(&prData)
			if err != nil {
				chErr <- fmt.Errorf("failed to scan pull request: %w", err)
				return
			}

			chPRs <- s.mapPullReq(ctx, &prData)
		}

		if err := rows.Err(); err != nil {
			chErr <- fmt.Errorf("failed to scan pull request: %w", err)
		}
	}()

	return chPRs, chErr
}

func (s *PullReqStore) ListOpenByBranchName(
	ctx context.Context,
	repoID int64,
	branchNames []string,
) (map[string][]*types.PullReq, error) {
	columns := pullReqColumnsNoDescription
	stmt := database.Builder.Select(columns)
	stmt = stmt.From("pullreqs")
	stmt = stmt.Where("pullreq_source_repo_id = ?", repoID)
	stmt = stmt.Where("pullreq_state = ?", enum.PullReqStateOpen)
	stmt = stmt.Where(squirrel.Eq{"pullreq_source_branch": branchNames})
	stmt = stmt.OrderBy("pullreq_updated desc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]*pullReq, 0)

	err = db.SelectContext(ctx, &dst, sql, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to fetch list of PRs by branch")
	}

	prMap := make(map[string][]*types.PullReq)
	for _, prDB := range dst {
		pr := s.mapPullReq(ctx, prDB)
		prMap[prDB.SourceBranch] = append(prMap[prDB.SourceBranch], pr)
	}

	return prMap, nil
}

func (s *PullReqStore) listQuery(opts *types.PullReqFilter) squirrel.SelectBuilder {
	var stmt squirrel.SelectBuilder

	columns := pullReqColumns
	if opts.ExcludeDescription {
		columns = pullReqColumnsNoDescription
	}

	if len(opts.LabelID) > 0 || len(opts.ValueID) > 0 || opts.CommenterID > 0 || opts.MentionedID > 0 {
		stmt = database.Builder.Select("DISTINCT " + columns)
	} else {
		stmt = database.Builder.Select(columns)
	}

	stmt = stmt.From("pullreqs")

	s.applyFilter(&stmt, opts)

	return stmt
}

//nolint:cyclop,gocognit,gocyclo
func (s *PullReqStore) applyFilter(stmt *squirrel.SelectBuilder, opts *types.PullReqFilter) {
	if len(opts.States) == 1 {
		*stmt = stmt.Where("pullreq_state = ?", opts.States[0])
	} else if len(opts.States) > 1 {
		*stmt = stmt.Where(squirrel.Eq{"pullreq_state": opts.States})
	}

	if opts.SourceRepoID != 0 {
		*stmt = stmt.Where("pullreq_source_repo_id = ?", opts.SourceRepoID)
	}

	if opts.SourceBranch != "" {
		*stmt = stmt.Where("pullreq_source_branch = ?", opts.SourceBranch)
	}

	if opts.TargetRepoID != 0 {
		*stmt = stmt.Where("pullreq_target_repo_id = ?", opts.TargetRepoID)
	}

	if opts.TargetBranch != "" {
		*stmt = stmt.Where("pullreq_target_branch = ?", opts.TargetBranch)
	}

	if opts.Query != "" {
		*stmt = stmt.Where(PartialMatch("pullreq_title", opts.Query))
	}

	if opts.CreatedLt > 0 {
		*stmt = stmt.Where("pullreq_created < ?", opts.CreatedLt)
	}

	if opts.CreatedGt > 0 {
		*stmt = stmt.Where("pullreq_created > ?", opts.CreatedGt)
	}

	if opts.UpdatedLt > 0 {
		*stmt = stmt.Where("pullreq_updated < ?", opts.UpdatedLt)
	}

	if opts.UpdatedGt > 0 {
		*stmt = stmt.Where("pullreq_updated > ?", opts.UpdatedGt)
	}

	if opts.EditedLt > 0 {
		*stmt = stmt.Where("pullreq_edited < ?", opts.EditedLt)
	}

	if opts.EditedGt > 0 {
		*stmt = stmt.Where("pullreq_edited > ?", opts.EditedGt)
	}

	if len(opts.SpaceIDs) == 1 {
		*stmt = stmt.InnerJoin("repositories ON repo_id = pullreq_target_repo_id")
		*stmt = stmt.Where("repo_parent_id = ?", opts.SpaceIDs[0])
	} else if len(opts.SpaceIDs) > 1 {
		*stmt = stmt.InnerJoin("repositories ON repo_id = pullreq_target_repo_id")
		switch s.db.DriverName() {
		case SqliteDriverName:
			*stmt = stmt.Where(squirrel.Eq{"repo_parent_id": opts.SpaceIDs})
		case PostgresDriverName:
			*stmt = stmt.Where("repo_parent_id = ANY(?)", pq.Array(opts.SpaceIDs))
		}
	}

	if len(opts.RepoIDBlacklist) == 1 {
		*stmt = stmt.Where("pullreq_target_repo_id <> ?", opts.RepoIDBlacklist[0])
	} else if len(opts.RepoIDBlacklist) > 1 {
		switch s.db.DriverName() {
		case SqliteDriverName:
			*stmt = stmt.Where(squirrel.NotEq{"pullreq_target_repo_id": opts.RepoIDBlacklist})
		case PostgresDriverName:
			*stmt = stmt.Where("pullreq_target_repo_id <> ALL(?)", pq.Array(opts.RepoIDBlacklist))
		}
	}

	if len(opts.CreatedBy) == 1 {
		*stmt = stmt.Where("pullreq_created_by = ?", opts.CreatedBy[0])
	} else if len(opts.CreatedBy) > 1 {
		switch s.db.DriverName() {
		case SqliteDriverName:
			*stmt = stmt.Where(squirrel.Eq{"pullreq_created_by": opts.CreatedBy})
		case PostgresDriverName:
			*stmt = stmt.Where("pullreq_created_by = ANY(?)", pq.Array(opts.CreatedBy))
		}
	}

	if opts.CommenterID > 0 {
		*stmt = stmt.InnerJoin("pullreq_activities act_com ON act_com.pullreq_activity_pullreq_id = pullreq_id")
		*stmt = stmt.Where("act_com.pullreq_activity_deleted IS NULL")
		*stmt = stmt.Where("(" +
			"act_com.pullreq_activity_kind = '" + string(enum.PullReqActivityKindComment) + "' OR " +
			"act_com.pullreq_activity_kind = '" + string(enum.PullReqActivityKindChangeComment) + "')")
		*stmt = stmt.Where("act_com.pullreq_activity_created_by = ?", opts.CommenterID)
	}

	if opts.ReviewerID > 0 {
		*stmt = stmt.InnerJoin(
			fmt.Sprintf("pullreq_reviewers ON "+
				"pullreq_reviewer_pullreq_id = pullreq_id AND pullreq_reviewer_principal_id = %d", opts.ReviewerID))
		if len(opts.ReviewDecisions) > 0 {
			*stmt = stmt.Where(squirrel.Eq{"pullreq_reviewer_review_decision": opts.ReviewDecisions})
		}
	}

	if opts.MentionedID > 0 {
		*stmt = stmt.InnerJoin("pullreq_activities act_ment ON act_ment.pullreq_activity_pullreq_id = pullreq_id")
		*stmt = stmt.Where("act_ment.pullreq_activity_deleted IS NULL")
		*stmt = stmt.Where("(" +
			"act_ment.pullreq_activity_kind = '" + string(enum.PullReqActivityKindComment) + "' OR " +
			"act_ment.pullreq_activity_kind = '" + string(enum.PullReqActivityKindChangeComment) + "')")

		switch s.db.DriverName() {
		case SqliteDriverName:
			*stmt = stmt.InnerJoin(
				"json_each(json_extract(act_ment.pullreq_activity_metadata, '$.mentions.ids')) as mentions")
			*stmt = stmt.Where("mentions.value = ?", opts.MentionedID)
		case PostgresDriverName:
			*stmt = stmt.Where(fmt.Sprintf(
				"act_ment.pullreq_activity_metadata->'mentions'->'ids' @> ('[%d]')::jsonb",
				opts.MentionedID))
		}
	}

	// labels

	if len(opts.LabelID) == 0 && len(opts.ValueID) == 0 {
		return
	}

	*stmt = stmt.InnerJoin("pullreq_labels ON pullreq_label_pullreq_id = pullreq_id").
		GroupBy("pullreq_id")

	switch {
	case len(opts.LabelID) > 0 && len(opts.ValueID) == 0:
		*stmt = stmt.Where(
			squirrel.Eq{"pullreq_label_label_id": opts.LabelID},
		)

	case len(opts.LabelID) == 0 && len(opts.ValueID) > 0:
		*stmt = stmt.Where(
			squirrel.Eq{"pullreq_label_label_value_id": opts.ValueID},
		)

	default:
		*stmt = stmt.Where(squirrel.Or{
			squirrel.Eq{"pullreq_label_label_id": opts.LabelID},
			squirrel.Eq{"pullreq_label_label_value_id": opts.ValueID},
		})
	}

	*stmt = stmt.Having("COUNT(pullreq_label_pullreq_id) = ?", len(opts.LabelID)+len(opts.ValueID))
}

func mapPullReq(pr *pullReq) *types.PullReq {
	var mergeConflicts, rebaseConflicts []string
	if pr.MergeConflicts.Valid {
		mergeConflicts = strings.Split(pr.MergeConflicts.String, "\n")
	}
	if pr.RebaseConflicts.Valid {
		rebaseConflicts = strings.Split(pr.RebaseConflicts.String, "\n")
	}

	return &types.PullReq{
		ID:                pr.ID,
		Version:           pr.Version,
		Number:            pr.Number,
		CreatedBy:         pr.CreatedBy,
		Created:           pr.Created,
		Updated:           pr.Updated,
		Edited:            pr.Edited, // TODO: When we remove the DB column, make Edited equal to Updated
		Closed:            pr.Closed.Ptr(),
		State:             pr.State,
		IsDraft:           pr.IsDraft,
		CommentCount:      pr.CommentCount,
		UnresolvedCount:   pr.UnresolvedCount,
		Title:             pr.Title,
		Description:       pr.Description,
		SourceRepoID:      pr.SourceRepoID,
		SourceBranch:      pr.SourceBranch,
		SourceSHA:         pr.SourceSHA,
		TargetRepoID:      pr.TargetRepoID,
		TargetBranch:      pr.TargetBranch,
		ActivitySeq:       pr.ActivitySeq,
		MergedBy:          pr.MergedBy.Ptr(),
		Merged:            pr.Merged.Ptr(),
		MergeMethod:       (*enum.MergeMethod)(pr.MergeMethod.Ptr()),
		MergeCheckStatus:  pr.MergeCheckStatus,
		MergeTargetSHA:    pr.MergeTargetSHA.Ptr(),
		MergeBaseSHA:      pr.MergeBaseSHA,
		MergeSHA:          pr.MergeSHA.Ptr(),
		MergeConflicts:    mergeConflicts,
		RebaseCheckStatus: pr.RebaseCheckStatus,
		RebaseConflicts:   rebaseConflicts,
		Author:            types.PrincipalInfo{},
		Merger:            nil,
		Stats: types.PullReqStats{
			Conversations:   pr.CommentCount,
			UnresolvedCount: pr.UnresolvedCount,
			DiffStats: types.DiffStats{
				Commits:      pr.CommitCount.Ptr(),
				FilesChanged: pr.FileCount.Ptr(),
				Additions:    pr.Additions.Ptr(),
				Deletions:    pr.Deletions.Ptr(),
			},
		},
	}
}

func mapInternalPullReq(pr *types.PullReq) *pullReq {
	mergeConflicts := strings.Join(pr.MergeConflicts, "\n")
	rebaseConflicts := strings.Join(pr.RebaseConflicts, "\n")
	m := &pullReq{
		ID:                pr.ID,
		Version:           pr.Version,
		Number:            pr.Number,
		CreatedBy:         pr.CreatedBy,
		Created:           pr.Created,
		Updated:           pr.Updated,
		Edited:            pr.Edited, // TODO: When we remove the DB column, make Edited equal to Updated
		Closed:            null.IntFromPtr(pr.Closed),
		State:             pr.State,
		IsDraft:           pr.IsDraft,
		CommentCount:      pr.CommentCount,
		UnresolvedCount:   pr.UnresolvedCount,
		Title:             pr.Title,
		Description:       pr.Description,
		SourceRepoID:      pr.SourceRepoID,
		SourceBranch:      pr.SourceBranch,
		SourceSHA:         pr.SourceSHA,
		TargetRepoID:      pr.TargetRepoID,
		TargetBranch:      pr.TargetBranch,
		ActivitySeq:       pr.ActivitySeq,
		MergedBy:          null.IntFromPtr(pr.MergedBy),
		Merged:            null.IntFromPtr(pr.Merged),
		MergeMethod:       null.StringFromPtr((*string)(pr.MergeMethod)),
		MergeCheckStatus:  pr.MergeCheckStatus,
		MergeTargetSHA:    null.StringFromPtr(pr.MergeTargetSHA),
		MergeBaseSHA:      pr.MergeBaseSHA,
		MergeSHA:          null.StringFromPtr(pr.MergeSHA),
		MergeConflicts:    null.NewString(mergeConflicts, mergeConflicts != ""),
		RebaseCheckStatus: pr.RebaseCheckStatus,
		RebaseConflicts:   null.NewString(rebaseConflicts, rebaseConflicts != ""),
		CommitCount:       null.IntFromPtr(pr.Stats.Commits),
		FileCount:         null.IntFromPtr(pr.Stats.FilesChanged),
		Additions:         null.IntFromPtr(pr.Stats.Additions),
		Deletions:         null.IntFromPtr(pr.Stats.Deletions),
	}

	return m
}

func (s *PullReqStore) mapPullReq(ctx context.Context, pr *pullReq) *types.PullReq {
	m := mapPullReq(pr)

	var author, merger *types.PrincipalInfo
	var err error

	author, err = s.pCache.Get(ctx, pr.CreatedBy)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to load PR author")
	}
	if author != nil {
		m.Author = *author
	}

	if pr.MergedBy.Valid {
		merger, err = s.pCache.Get(ctx, pr.MergedBy.Int64)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to load PR merger")
		}
		m.Merger = merger
	}

	return m
}

func (s *PullReqStore) mapSlicePullReq(ctx context.Context, prs []*pullReq) ([]*types.PullReq, error) {
	// collect all principal IDs
	ids := make([]int64, 0, 2*len(prs))
	for _, pr := range prs {
		ids = append(ids, pr.CreatedBy)
		if pr.MergedBy.Valid {
			ids = append(ids, pr.MergedBy.Int64)
		}
	}

	// pull principal infos from cache
	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load PR principal infos: %w", err)
	}

	// attach the principal infos back to the slice items
	m := make([]*types.PullReq, len(prs))
	for i, pr := range prs {
		m[i] = mapPullReq(pr)
		if author, ok := infoMap[pr.CreatedBy]; ok {
			m[i].Author = *author
		}
		if pr.MergedBy.Valid {
			if merger, ok := infoMap[pr.MergedBy.Int64]; ok {
				m[i].Merger = merger
			}
		}
	}

	return m, nil
}
