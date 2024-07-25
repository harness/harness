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
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ store.PullReqActivityStore = (*PullReqActivityStore)(nil)

// NewPullReqActivityStore returns a new PullReqJournalStore.
func NewPullReqActivityStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
) *PullReqActivityStore {
	return &PullReqActivityStore{
		db:     db,
		pCache: pCache,
	}
}

// PullReqActivityStore implements store.PullReqActivityStore backed by a relational database.
type PullReqActivityStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

// journal is used to fetch pull request data from the database.
// The object should be later re-packed into a different struct to return it as an API response.
type pullReqActivity struct {
	ID      int64 `db:"pullreq_activity_id"`
	Version int64 `db:"pullreq_activity_version"`

	CreatedBy int64    `db:"pullreq_activity_created_by"`
	Created   int64    `db:"pullreq_activity_created"`
	Updated   int64    `db:"pullreq_activity_updated"`
	Edited    int64    `db:"pullreq_activity_edited"`
	Deleted   null.Int `db:"pullreq_activity_deleted"`

	ParentID  null.Int `db:"pullreq_activity_parent_id"`
	RepoID    int64    `db:"pullreq_activity_repo_id"`
	PullReqID int64    `db:"pullreq_activity_pullreq_id"`

	Order    int64 `db:"pullreq_activity_order"`
	SubOrder int64 `db:"pullreq_activity_sub_order"`
	ReplySeq int64 `db:"pullreq_activity_reply_seq"`

	Type enum.PullReqActivityType `db:"pullreq_activity_type"`
	Kind enum.PullReqActivityKind `db:"pullreq_activity_kind"`

	Text     string          `db:"pullreq_activity_text"`
	Payload  json.RawMessage `db:"pullreq_activity_payload"`
	Metadata json.RawMessage `db:"pullreq_activity_metadata"`

	ResolvedBy null.Int `db:"pullreq_activity_resolved_by"`
	Resolved   null.Int `db:"pullreq_activity_resolved"`

	Outdated                null.Bool   `db:"pullreq_activity_outdated"`
	CodeCommentMergeBaseSHA null.String `db:"pullreq_activity_code_comment_merge_base_sha"`
	CodeCommentSourceSHA    null.String `db:"pullreq_activity_code_comment_source_sha"`
	CodeCommentPath         null.String `db:"pullreq_activity_code_comment_path"`
	CodeCommentLineNew      null.Int    `db:"pullreq_activity_code_comment_line_new"`
	CodeCommentSpanNew      null.Int    `db:"pullreq_activity_code_comment_span_new"`
	CodeCommentLineOld      null.Int    `db:"pullreq_activity_code_comment_line_old"`
	CodeCommentSpanOld      null.Int    `db:"pullreq_activity_code_comment_span_old"`
}

const (
	pullreqActivityColumns = `
		 pullreq_activity_id
		,pullreq_activity_version
		,pullreq_activity_created_by
		,pullreq_activity_created
		,pullreq_activity_updated
		,pullreq_activity_edited
		,pullreq_activity_deleted
		,pullreq_activity_parent_id
		,pullreq_activity_repo_id
		,pullreq_activity_pullreq_id
		,pullreq_activity_order
		,pullreq_activity_sub_order
		,pullreq_activity_reply_seq
		,pullreq_activity_type
		,pullreq_activity_kind
		,pullreq_activity_text
		,pullreq_activity_payload
		,pullreq_activity_metadata
		,pullreq_activity_resolved_by
		,pullreq_activity_resolved
		,pullreq_activity_outdated
		,pullreq_activity_code_comment_merge_base_sha
		,pullreq_activity_code_comment_source_sha
		,pullreq_activity_code_comment_path
		,pullreq_activity_code_comment_line_new
		,pullreq_activity_code_comment_span_new
		,pullreq_activity_code_comment_line_old
		,pullreq_activity_code_comment_span_old`

	pullreqActivitySelectBase = `
	SELECT` + pullreqActivityColumns + `
	FROM pullreq_activities`
)

// Find finds the pull request activity by id.
func (s *PullReqActivityStore) Find(ctx context.Context, id int64) (*types.PullReqActivity, error) {
	const sqlQuery = pullreqActivitySelectBase + `
	WHERE pullreq_activity_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReqActivity{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request activity")
	}

	act, err := s.mapPullReqActivity(ctx, dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map pull request activity: %w", err)
	}

	return act, nil
}

// Create creates a new pull request.
func (s *PullReqActivityStore) Create(ctx context.Context, act *types.PullReqActivity) error {
	const sqlQuery = `
	INSERT INTO pullreq_activities (
		 pullreq_activity_version
		,pullreq_activity_created_by
		,pullreq_activity_created
		,pullreq_activity_updated
		,pullreq_activity_edited
		,pullreq_activity_deleted
		,pullreq_activity_parent_id
		,pullreq_activity_repo_id
		,pullreq_activity_pullreq_id
		,pullreq_activity_order
		,pullreq_activity_sub_order
		,pullreq_activity_reply_seq
		,pullreq_activity_type
		,pullreq_activity_kind
		,pullreq_activity_text
		,pullreq_activity_payload
		,pullreq_activity_metadata
		,pullreq_activity_resolved_by
		,pullreq_activity_resolved
		,pullreq_activity_outdated
		,pullreq_activity_code_comment_merge_base_sha
		,pullreq_activity_code_comment_source_sha
		,pullreq_activity_code_comment_path
		,pullreq_activity_code_comment_line_new
		,pullreq_activity_code_comment_span_new
		,pullreq_activity_code_comment_line_old
		,pullreq_activity_code_comment_span_old
	) values (
		 :pullreq_activity_version
		,:pullreq_activity_created_by
		,:pullreq_activity_created
		,:pullreq_activity_updated
		,:pullreq_activity_edited
		,:pullreq_activity_deleted
		,:pullreq_activity_parent_id
		,:pullreq_activity_repo_id
		,:pullreq_activity_pullreq_id
		,:pullreq_activity_order
		,:pullreq_activity_sub_order
		,:pullreq_activity_reply_seq
		,:pullreq_activity_type
		,:pullreq_activity_kind
		,:pullreq_activity_text
		,:pullreq_activity_payload
		,:pullreq_activity_metadata
		,:pullreq_activity_resolved_by
		,:pullreq_activity_resolved
		,:pullreq_activity_outdated
		,:pullreq_activity_code_comment_merge_base_sha
		,:pullreq_activity_code_comment_source_sha
		,:pullreq_activity_code_comment_path
		,:pullreq_activity_code_comment_line_new
		,:pullreq_activity_code_comment_span_new
		,:pullreq_activity_code_comment_line_old
		,:pullreq_activity_code_comment_span_old
	) RETURNING pullreq_activity_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbAct, err := mapInternalPullReqActivity(act)
	if err != nil {
		return fmt.Errorf("failed to map pull request activity: %w", err)
	}

	query, arg, err := db.BindNamed(sqlQuery, dbAct)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request activity object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&act.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert pull request activity")
	}

	return nil
}

func (s *PullReqActivityStore) CreateWithPayload(ctx context.Context,
	pr *types.PullReq, principalID int64, payload types.PullReqActivityPayload, metadata *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		CreatedBy: principalID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     pr.ActivitySeq,
		SubOrder:  0,
		ReplySeq:  0,
		Type:      payload.ActivityType(),
		Kind:      enum.PullReqActivityKindSystem,
		Text:      "",
		Metadata:  metadata,
	}

	_ = act.SetPayload(payload)

	err := s.Create(ctx, act)
	if err != nil {
		err = fmt.Errorf("failed to write pull request system '%s' activity: %w", payload.ActivityType(), err)
		return nil, err
	}

	return act, nil
}

// Update updates the pull request.
func (s *PullReqActivityStore) Update(ctx context.Context, act *types.PullReqActivity) error {
	const sqlQuery = `
	UPDATE pullreq_activities
	SET
	     pullreq_activity_version = :pullreq_activity_version
		,pullreq_activity_updated = :pullreq_activity_updated
		,pullreq_activity_edited = :pullreq_activity_edited
		,pullreq_activity_deleted = :pullreq_activity_deleted
		,pullreq_activity_reply_seq = :pullreq_activity_reply_seq
		,pullreq_activity_text = :pullreq_activity_text
		,pullreq_activity_payload = :pullreq_activity_payload
		,pullreq_activity_metadata = :pullreq_activity_metadata
		,pullreq_activity_resolved_by = :pullreq_activity_resolved_by
		,pullreq_activity_resolved = :pullreq_activity_resolved
		,pullreq_activity_outdated = :pullreq_activity_outdated
		,pullreq_activity_code_comment_merge_base_sha = :pullreq_activity_code_comment_merge_base_sha
		,pullreq_activity_code_comment_source_sha = :pullreq_activity_code_comment_source_sha
		,pullreq_activity_code_comment_path = :pullreq_activity_code_comment_path
		,pullreq_activity_code_comment_line_new = :pullreq_activity_code_comment_line_new
		,pullreq_activity_code_comment_span_new = :pullreq_activity_code_comment_span_new
		,pullreq_activity_code_comment_line_old = :pullreq_activity_code_comment_line_old
		,pullreq_activity_code_comment_span_old = :pullreq_activity_code_comment_span_old
	WHERE pullreq_activity_id = :pullreq_activity_id AND pullreq_activity_version = :pullreq_activity_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbAct, err := mapInternalPullReqActivity(act)
	if err != nil {
		return fmt.Errorf("failed to map pull request activity: %w", err)
	}
	dbAct.Version++
	dbAct.Updated = updatedAt.UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbAct)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request activity object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update pull request activity")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	updatedAct, err := s.mapPullReqActivity(ctx, dbAct)
	if err != nil {
		return fmt.Errorf("failed to map db pull request activity: %w", err)
	}
	*act = *updatedAct

	return nil
}

// UpdateOptLock updates the pull request using the optimistic locking mechanism.
func (s *PullReqActivityStore) UpdateOptLock(ctx context.Context,
	act *types.PullReqActivity,
	mutateFn func(act *types.PullReqActivity) error,
) (*types.PullReqActivity, error) {
	for {
		dup := *act

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

		act, err = s.Find(ctx, act.ID)
		if err != nil {
			return nil, err
		}
	}
}

// Count of pull requests for a repo.
func (s *PullReqActivityStore) Count(ctx context.Context,
	prID int64,
	opts *types.PullReqActivityFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID)

	if len(opts.Types) == 1 {
		stmt = stmt.Where("pullreq_activity_type = ?", opts.Types[0])
	} else if len(opts.Types) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_type": opts.Types})
	}

	if len(opts.Kinds) == 1 {
		stmt = stmt.Where("pullreq_activity_kind = ?", opts.Kinds[0])
	} else if len(opts.Kinds) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_kind": opts.Kinds})
	}

	if opts.After != 0 {
		stmt = stmt.Where("pullreq_activity_created > ?", opts.After)
	}

	if opts.Before != 0 {
		stmt = stmt.Where("pullreq_activity_created < ?", opts.Before)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of pull request activities for a PR.
func (s *PullReqActivityStore) List(ctx context.Context,
	prID int64,
	filter *types.PullReqActivityFilter,
) ([]*types.PullReqActivity, error) {
	stmt := database.Builder.
		Select(pullreqActivityColumns).
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID)

	stmt = applyFilter(filter, stmt)

	stmt = stmt.OrderBy("pullreq_activity_order asc", "pullreq_activity_sub_order asc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request activity query to sql")
	}

	dst := make([]*pullReqActivity, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing pull request activity list query")
	}

	result, err := s.mapSlicePullReqActivity(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListAuthorIDs returns a list of pull request activity author ids in a thread for a PR.
func (s *PullReqActivityStore) ListAuthorIDs(ctx context.Context, prID int64, order int64) ([]int64, error) {
	stmt := database.Builder.
		Select("DISTINCT pullreq_activity_created_by").
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID).
		Where("pullreq_activity_order = ?", order)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request activity query to sql")
	}

	var dst []int64

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing pull request activity list query")
	}

	return dst, nil
}

func (s *PullReqActivityStore) CountUnresolved(ctx context.Context, prID int64) (int, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID).
		Where("pullreq_activity_sub_order = 0").
		Where("pullreq_activity_resolved IS NULL").
		Where("pullreq_activity_deleted IS NULL").
		Where("pullreq_activity_kind <> ?", enum.PullReqActivityKindSystem)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count unresolved query")
	}

	return count, nil
}

func mapPullReqActivity(act *pullReqActivity) (*types.PullReqActivity, error) {
	metadata := &types.PullReqActivityMetadata{}
	err := json.Unmarshal(act.Metadata, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	m := &types.PullReqActivity{
		ID:         act.ID,
		Version:    act.Version,
		CreatedBy:  act.CreatedBy,
		Created:    act.Created,
		Updated:    act.Updated,
		Edited:     act.Edited,
		Deleted:    act.Deleted.Ptr(),
		ParentID:   act.ParentID.Ptr(),
		RepoID:     act.RepoID,
		PullReqID:  act.PullReqID,
		Order:      act.Order,
		SubOrder:   act.SubOrder,
		ReplySeq:   act.ReplySeq,
		Type:       act.Type,
		Kind:       act.Kind,
		Text:       act.Text,
		PayloadRaw: act.Payload,
		Metadata:   metadata,
		ResolvedBy: act.ResolvedBy.Ptr(),
		Resolved:   act.Resolved.Ptr(),
		Author:     types.PrincipalInfo{},
		Resolver:   nil,
	}
	if m.Type == enum.PullReqActivityTypeCodeComment && m.Kind == enum.PullReqActivityKindChangeComment {
		m.CodeComment = &types.CodeCommentFields{
			Outdated:     act.Outdated.Bool,
			MergeBaseSHA: act.CodeCommentMergeBaseSHA.String,
			SourceSHA:    act.CodeCommentSourceSHA.String,
			Path:         act.CodeCommentPath.String,
			LineNew:      int(act.CodeCommentLineNew.Int64),
			SpanNew:      int(act.CodeCommentSpanNew.Int64),
			LineOld:      int(act.CodeCommentLineOld.Int64),
			SpanOld:      int(act.CodeCommentSpanOld.Int64),
		}
	}

	return m, nil
}

func mapInternalPullReqActivity(act *types.PullReqActivity) (*pullReqActivity, error) {
	m := &pullReqActivity{
		ID:         act.ID,
		Version:    act.Version,
		CreatedBy:  act.CreatedBy,
		Created:    act.Created,
		Updated:    act.Updated,
		Edited:     act.Edited,
		Deleted:    null.IntFromPtr(act.Deleted),
		ParentID:   null.IntFromPtr(act.ParentID),
		RepoID:     act.RepoID,
		PullReqID:  act.PullReqID,
		Order:      act.Order,
		SubOrder:   act.SubOrder,
		ReplySeq:   act.ReplySeq,
		Type:       act.Type,
		Kind:       act.Kind,
		Text:       act.Text,
		Payload:    act.PayloadRaw,
		Metadata:   nil,
		ResolvedBy: null.IntFromPtr(act.ResolvedBy),
		Resolved:   null.IntFromPtr(act.Resolved),
	}
	if act.IsValidCodeComment() {
		m.Outdated = null.BoolFrom(act.CodeComment.Outdated)
		m.CodeCommentMergeBaseSHA = null.StringFrom(act.CodeComment.MergeBaseSHA)
		m.CodeCommentSourceSHA = null.StringFrom(act.CodeComment.SourceSHA)
		m.CodeCommentPath = null.StringFrom(act.CodeComment.Path)
		m.CodeCommentLineNew = null.IntFrom(int64(act.CodeComment.LineNew))
		m.CodeCommentSpanNew = null.IntFrom(int64(act.CodeComment.SpanNew))
		m.CodeCommentLineOld = null.IntFrom(int64(act.CodeComment.LineOld))
		m.CodeCommentSpanOld = null.IntFrom(int64(act.CodeComment.SpanOld))
	}

	var err error
	m.Metadata, err = json.Marshal(act.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return m, nil
}

func (s *PullReqActivityStore) mapPullReqActivity(
	ctx context.Context,
	act *pullReqActivity,
) (*types.PullReqActivity, error) {
	m, err := mapPullReqActivity(act)
	if err != nil {
		return nil, err
	}

	var author, resolver *types.PrincipalInfo

	author, err = s.pCache.Get(ctx, act.CreatedBy)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to load PR activity author")
	}
	if author != nil {
		m.Author = *author
	}

	if act.ResolvedBy.Valid {
		resolver, err = s.pCache.Get(ctx, act.ResolvedBy.Int64)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to load PR activity resolver")
		}
		m.Resolver = resolver
	}

	return m, nil
}

func (s *PullReqActivityStore) mapSlicePullReqActivity(
	ctx context.Context,
	activities []*pullReqActivity,
) ([]*types.PullReqActivity, error) {
	// collect all principal IDs
	ids := make([]int64, 0, 2*len(activities))
	for _, act := range activities {
		ids = append(ids, act.CreatedBy)
		if act.ResolvedBy.Valid {
			ids = append(ids, act.ResolvedBy.Int64)
		}
	}

	// pull principal infos from cache
	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load PR principal infos: %w", err)
	}

	// attach the principal infos back to the slice items
	m := make([]*types.PullReqActivity, len(activities))
	for i, act := range activities {
		m[i], err = mapPullReqActivity(act)
		if err != nil {
			return nil, fmt.Errorf("failed to map pull request activity %d: %w", act.ID, err)
		}
		if author, ok := infoMap[act.CreatedBy]; ok {
			m[i].Author = *author
		}
		if act.ResolvedBy.Valid {
			if merger, ok := infoMap[act.ResolvedBy.Int64]; ok {
				m[i].Resolver = merger
			}
		}
	}

	return m, nil
}

func applyFilter(
	filter *types.PullReqActivityFilter,
	stmt squirrel.SelectBuilder,
) squirrel.SelectBuilder {
	if len(filter.Types) == 1 {
		stmt = stmt.Where("pullreq_activity_type = ?", filter.Types[0])
	} else if len(filter.Types) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_type": filter.Types})
	}

	if len(filter.Kinds) == 1 {
		stmt = stmt.Where("pullreq_activity_kind = ?", filter.Kinds[0])
	} else if len(filter.Kinds) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_kind": filter.Kinds})
	}

	if filter.After != 0 {
		stmt = stmt.Where("pullreq_activity_created > ?", filter.After)
	}

	if filter.Before != 0 {
		stmt = stmt.Where("pullreq_activity_created < ?", filter.Before)
	}

	if filter.Limit > 0 {
		stmt = stmt.Limit(database.Limit(filter.Limit))
	}

	return stmt
}
