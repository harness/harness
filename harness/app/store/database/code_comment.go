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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ store.CodeCommentView = (*CodeCommentView)(nil)

// NewCodeCommentView returns a new CodeCommentView.
func NewCodeCommentView(db *sqlx.DB) *CodeCommentView {
	return &CodeCommentView{
		db: db,
	}
}

// CodeCommentView implements store.CodeCommentView backed by a relational database.
type CodeCommentView struct {
	db *sqlx.DB
}

// ListNotAtSourceSHA lists all code comments not already at the provided source SHA.
func (s *CodeCommentView) ListNotAtSourceSHA(ctx context.Context,
	prID int64, sourceSHA string,
) ([]*types.CodeComment, error) {
	return s.list(ctx, prID, "", sourceSHA)
}

// ListNotAtMergeBaseSHA lists all code comments not already at the provided merge base SHA.
func (s *CodeCommentView) ListNotAtMergeBaseSHA(ctx context.Context,
	prID int64, mergeBaseSHA string,
) ([]*types.CodeComment, error) {
	return s.list(ctx, prID, mergeBaseSHA, "")
}

// list is used by internal service that updates line numbers of code comments after
// branch updates and requires either mergeBaseSHA or sourceSHA but not both.
// Resulting list is ordered by the file name and the relevant line number.
func (s *CodeCommentView) list(ctx context.Context,
	prID int64, mergeBaseSHA, sourceSHA string,
) ([]*types.CodeComment, error) {
	const codeCommentColumns = `
		 pullreq_activity_id
		,pullreq_activity_version
		,pullreq_activity_updated
		,coalesce(pullreq_activity_outdated, false) as "pullreq_activity_outdated"
		,coalesce(pullreq_activity_code_comment_merge_base_sha, '') as "pullreq_activity_code_comment_merge_base_sha"
		,coalesce(pullreq_activity_code_comment_source_sha, '') as "pullreq_activity_code_comment_source_sha"
		,coalesce(pullreq_activity_code_comment_path, '') as "pullreq_activity_code_comment_path"
		,coalesce(pullreq_activity_code_comment_line_new, 1) as "pullreq_activity_code_comment_line_new"
		,coalesce(pullreq_activity_code_comment_span_new, 0) as "pullreq_activity_code_comment_span_new"
		,coalesce(pullreq_activity_code_comment_line_old, 1) as "pullreq_activity_code_comment_line_old"
		,coalesce(pullreq_activity_code_comment_span_old, 0) as "pullreq_activity_code_comment_span_old"`

	stmt := database.Builder.
		Select(codeCommentColumns).
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID).
		Where("not pullreq_activity_outdated").
		Where("pullreq_activity_type = ?", enum.PullReqActivityTypeCodeComment).
		Where("pullreq_activity_kind = ?", enum.PullReqActivityKindChangeComment).
		Where("pullreq_activity_deleted is null and pullreq_activity_parent_id is null")

	if mergeBaseSHA != "" {
		stmt = stmt.
			Where("pullreq_activity_code_comment_merge_base_sha <> ?", mergeBaseSHA)
	} else {
		stmt = stmt.
			Where("pullreq_activity_code_comment_source_sha <> ?", sourceSHA)
	}

	stmt = stmt.OrderBy("pullreq_activity_code_comment_path asc",
		"pullreq_activity_code_comment_line_new asc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request activity query to sql")
	}

	result := make([]*types.CodeComment, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &result, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing code comment list query")
	}

	return result, nil
}

// UpdateAll updates all code comments provided in the slice.
func (s *CodeCommentView) UpdateAll(ctx context.Context, codeComments []*types.CodeComment) error {
	if len(codeComments) == 0 {
		return nil
	}

	const sqlQuery = `
	UPDATE pullreq_activities
	SET
	     pullreq_activity_version = :pullreq_activity_version
		,pullreq_activity_updated = :pullreq_activity_updated
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

	//nolint:sqlclosecheck
	stmt, err := db.PrepareNamedContext(ctx, sqlQuery)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to prepare update statement for update code comments")
	}

	updatedAt := time.Now()

	for _, codeComment := range codeComments {
		codeComment.Version++
		codeComment.Updated = updatedAt.UnixMilli()

		result, err := stmt.ExecContext(ctx, codeComment)
		if err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Failed to update code comment=%d", codeComment.ID)
		}

		count, err := result.RowsAffected()
		if err != nil {
			return database.ProcessSQLErrorf(
				ctx,
				err,
				"Failed to get number of updated rows for code comment=%d",
				codeComment.ID,
			)
		}

		if count == 0 {
			log.Ctx(ctx).Warn().Msgf("Version conflict when trying to update code comment=%d", codeComment.ID)
			continue
		}
	}

	return nil
}
