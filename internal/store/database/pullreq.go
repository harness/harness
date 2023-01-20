// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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

	CreatedBy int64 `db:"pullreq_created_by"`
	Created   int64 `db:"pullreq_created"`
	Updated   int64 `db:"pullreq_updated"`
	Edited    int64 `db:"pullreq_edited"`

	State   enum.PullReqState `db:"pullreq_state"`
	IsDraft bool              `db:"pullreq_is_draft"`

	Title       string `db:"pullreq_title"`
	Description string `db:"pullreq_description"`

	SourceRepoID int64  `db:"pullreq_source_repo_id"`
	SourceBranch string `db:"pullreq_source_branch"`
	TargetRepoID int64  `db:"pullreq_target_repo_id"`
	TargetBranch string `db:"pullreq_target_branch"`

	ActivitySeq int64 `db:"pullreq_activity_seq"`

	MergedBy      null.Int    `db:"pullreq_merged_by"`
	Merged        null.Int    `db:"pullreq_merged"`
	MergeStrategy null.String `db:"pullreq_merge_strategy"`
	MergeHeadSHA  null.String `db:"pullreq_merge_head_sha"`
	MergeBaseSHA  null.String `db:"pullreq_merge_base_sha"`
}

const (
	pullReqColumns = `
		 pullreq_id
		,pullreq_version
		,pullreq_number
		,pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_edited
		,pullreq_state
		,pullreq_is_draft
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_activity_seq
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_strategy
		,pullreq_merge_head_sha
		,pullreq_merge_base_sha`

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
		return nil, processSQLErrorf(err, "Failed to find pull request")
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
		sqlQuery += "\n" + sqlForUpdate
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReq{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, number); err != nil {
		return nil, processSQLErrorf(err, "Failed to find pull request by number")
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
		,pullreq_state
		,pullreq_is_draft
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_activity_seq
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_strategy
	) values (
		 :pullreq_version
		,:pullreq_number
		,:pullreq_created_by
		,:pullreq_created
		,:pullreq_updated
		,:pullreq_edited
		,:pullreq_state
		,:pullreq_is_draft
		,:pullreq_title
		,:pullreq_description
		,:pullreq_source_repo_id
		,:pullreq_source_branch
		,:pullreq_target_repo_id
		,:pullreq_target_branch
		,:pullreq_activity_seq
		,:pullreq_merged_by
		,:pullreq_merged
		,:pullreq_merge_strategy
	) RETURNING pullreq_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReq(pr))
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pullReq object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&pr.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
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
		,pullreq_state = :pullreq_state
		,pullreq_is_draft = :pullreq_is_draft
		,pullreq_title = :pullreq_title
		,pullreq_description = :pullreq_description
		,pullreq_activity_seq = :pullreq_activity_seq
		,pullreq_merged_by = :pullreq_merged_by
		,pullreq_merged = :pullreq_merged
		,pullreq_merge_strategy = :pullreq_merge_strategy
		,pullreq_merge_head_sha = :pullreq_merge_head_sha
		,pullreq_merge_base_sha = :pullreq_merge_base_sha
	WHERE pullreq_id = :pullreq_id AND pullreq_version = :pullreq_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbPR := mapInternalPullReq(pr)
	dbPR.Version++
	dbPR.Updated = updatedAt.UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbPR)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to update pull request")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return processSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return store.ErrVersionConflict
	}

	pr.Version = dbPR.Version
	pr.Updated = dbPR.Updated

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
		if !errors.Is(err, store.ErrVersionConflict) {
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

// Delete the pull request.
func (s *PullReqStore) Delete(ctx context.Context, id int64) error {
	const pullReqDelete = `DELETE FROM pullreqs WHERE pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, pullReqDelete, id); err != nil {
		return processSQLErrorf(err, "the delete query failed")
	}

	return nil
}

// Count of pull requests for a repo.
func (s *PullReqStore) Count(ctx context.Context, opts *types.PullReqFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("pullreqs")

	if len(opts.States) == 1 {
		stmt = stmt.Where("pullreq_state = ?", opts.States[0])
	} else if len(opts.States) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_state": opts.States})
	}

	if opts.SourceRepoID != 0 {
		stmt = stmt.Where("pullreq_source_repo_id = ?", opts.SourceRepoID)
	}

	if opts.SourceBranch != "" {
		stmt = stmt.Where("pullreq_source_branch = ?", opts.SourceBranch)
	}

	if opts.TargetRepoID != 0 {
		stmt = stmt.Where("pullreq_target_repo_id = ?", opts.TargetRepoID)
	}

	if opts.TargetBranch != "" {
		stmt = stmt.Where("pullreq_target_branch = ?", opts.TargetBranch)
	}

	if opts.CreatedBy != 0 {
		stmt = stmt.Where("pullreq_created_by = ?", opts.CreatedBy)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of pull requests for a repo.
func (s *PullReqStore) List(ctx context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
	stmt := builder.
		Select(pullReqColumns).
		From("pullreqs")

	if len(opts.States) == 1 {
		stmt = stmt.Where("pullreq_state = ?", opts.States[0])
	} else if len(opts.States) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_state": opts.States})
	}

	if opts.SourceRepoID != 0 {
		stmt = stmt.Where("pullreq_source_repo_id = ?", opts.SourceRepoID)
	}

	if opts.SourceBranch != "" {
		stmt = stmt.Where("pullreq_source_branch = ?", opts.SourceBranch)
	}

	if opts.TargetRepoID != 0 {
		stmt = stmt.Where("pullreq_target_repo_id = ?", opts.TargetRepoID)
	}

	if opts.TargetBranch != "" {
		stmt = stmt.Where("pullreq_target_branch = ?", opts.TargetBranch)
	}

	if opts.Query != "" {
		stmt = stmt.Where("pullreq_title LIKE ?", "%"+opts.Query+"%")
	}

	if opts.CreatedBy > 0 {
		stmt = stmt.Where("pullreq_created_by = ?", opts.CreatedBy)
	}

	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	// NOTE: string concatenation is safe because the
	// order attribute is an enum and is not user-defined,
	// and is therefore not subject to injection attacks.
	opts.Sort, _ = opts.Sort.Sanitize()
	stmt = stmt.OrderBy("pullreq_" + string(opts.Sort) + " " + opts.Order.String())

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	dst := make([]*pullReq, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	result, err := s.mapSlicePullReq(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func mapPullReq(pr *pullReq) *types.PullReq {
	return &types.PullReq{
		ID:            pr.ID,
		Version:       pr.Version,
		Number:        pr.Number,
		CreatedBy:     pr.CreatedBy,
		Created:       pr.Created,
		Updated:       pr.Updated,
		Edited:        pr.Edited,
		State:         pr.State,
		IsDraft:       pr.IsDraft,
		Title:         pr.Title,
		Description:   pr.Description,
		SourceRepoID:  pr.SourceRepoID,
		SourceBranch:  pr.SourceBranch,
		TargetRepoID:  pr.TargetRepoID,
		TargetBranch:  pr.TargetBranch,
		ActivitySeq:   pr.ActivitySeq,
		MergedBy:      pr.MergedBy.Ptr(),
		Merged:        pr.Merged.Ptr(),
		MergeStrategy: (*enum.MergeMethod)(pr.MergeStrategy.Ptr()),
		MergeHeadSHA:  pr.MergeHeadSHA.Ptr(),
		MergeBaseSHA:  pr.MergeBaseSHA.Ptr(),
		Author:        types.PrincipalInfo{},
		Merger:        nil,
	}
}

func mapInternalPullReq(pr *types.PullReq) *pullReq {
	m := &pullReq{
		ID:            pr.ID,
		Version:       pr.Version,
		Number:        pr.Number,
		CreatedBy:     pr.CreatedBy,
		Created:       pr.Created,
		Updated:       pr.Updated,
		Edited:        pr.Edited,
		State:         pr.State,
		IsDraft:       pr.IsDraft,
		Title:         pr.Title,
		Description:   pr.Description,
		SourceRepoID:  pr.SourceRepoID,
		SourceBranch:  pr.SourceBranch,
		TargetRepoID:  pr.TargetRepoID,
		TargetBranch:  pr.TargetBranch,
		ActivitySeq:   pr.ActivitySeq,
		MergedBy:      null.IntFromPtr(pr.MergedBy),
		Merged:        null.IntFromPtr(pr.Merged),
		MergeStrategy: null.StringFromPtr((*string)(pr.MergeStrategy)),
		MergeHeadSHA:  null.StringFromPtr(pr.MergeHeadSHA),
		MergeBaseSHA:  null.StringFromPtr(pr.MergeBaseSHA),
	}

	return m
}

func (s *PullReqStore) mapPullReq(ctx context.Context, pr *pullReq) *types.PullReq {
	m := mapPullReq(pr)

	var author, merger *types.PrincipalInfo
	var err error

	author, err = s.pCache.Get(ctx, pr.CreatedBy)
	if err != nil {
		log.Err(err).Msg("failed to load PR author")
	}
	if author != nil {
		m.Author = *author
	}

	if pr.MergedBy.Valid {
		merger, err = s.pCache.Get(ctx, pr.MergedBy.Int64)
		if err != nil {
			log.Err(err).Msg("failed to load PR merger")
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
