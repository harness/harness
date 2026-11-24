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
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.PullReqReviewerSuggestionStore = (*pullReqReviewerSuggestionStore)(nil)

func NewPullReqReviewerSuggestionStore(db *sqlx.DB) store.PullReqReviewerSuggestionStore {
	return &pullReqReviewerSuggestionStore{db: db}
}

type pullReqReviewerSuggestionStore struct {
	db *sqlx.DB
}

type pullReqReviewerSuggestion struct {
	PullReqID   int64 `db:"pullreq_reviewer_suggestion_pullreq_id"`
	CreatedBy   int64 `db:"pullreq_reviewer_suggestion_created_by"`
	PrincipalID int64 `db:"pullreq_reviewer_suggestion_principal_id"`
	Created     int64 `db:"pullreq_reviewer_suggestion_created"`

	Reason      null.String `db:"pullreq_reviewer_suggestion_reason"`
	Kind        string      `db:"pullreq_reviewer_suggestion_kind"`
	Rank        null.Int    `db:"pullreq_reviewer_suggestion_rank"`
	UserGroupID null.Int    `db:"pullreq_reviewer_suggestion_user_group_id"`
	CommitSHA   null.String `db:"pullreq_reviewer_suggestion_commit_sha"`
}

const pullReqReviewerSuggestionColumns = `
	pullreq_reviewer_suggestion_pullreq_id
	,pullreq_reviewer_suggestion_created_by
	,pullreq_reviewer_suggestion_principal_id
	,pullreq_reviewer_suggestion_created
	,pullreq_reviewer_suggestion_reason
	,pullreq_reviewer_suggestion_kind
	,pullreq_reviewer_suggestion_rank
	,pullreq_reviewer_suggestion_user_group_id
	,pullreq_reviewer_suggestion_commit_sha`

// Find returns reviewer suggestion by pull request id and principal id.
func (s *pullReqReviewerSuggestionStore) Find(
	ctx context.Context,
	prID, principalID int64,
) (*types.PullReqReviewerSuggestion, error) {
	stmt := database.Builder.
		Select(pullReqReviewerSuggestionColumns).
		From("pullreq_reviewer_suggestions").
		Where(squirrel.Eq{
			"pullreq_reviewer_suggestion_pullreq_id":   prID,
			"pullreq_reviewer_suggestion_principal_id": principalID,
		})

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)
	dst := new(pullReqReviewerSuggestion)

	if err = db.GetContext(ctx, dst, sqlQuery, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to find reviewer suggestion")
	}

	return mapPullReqReviewerSuggestion(dst), nil
}

// List returns all reviewer suggestions for a pull request.
func (s *pullReqReviewerSuggestionStore) List(
	ctx context.Context,
	prID int64,
	pagination types.Pagination,
) ([]*types.PullReqReviewerSuggestion, error) {
	stmt := database.Builder.
		Select(pullReqReviewerSuggestionColumns).
		From("pullreq_reviewer_suggestions").
		Where("pullreq_reviewer_suggestion_pullreq_id = ?", prID).
		OrderBy(
			"pullreq_reviewer_suggestion_rank IS NULL",
			"pullreq_reviewer_suggestion_rank ASC",
			"pullreq_reviewer_suggestion_created ASC",
			// A batch stamps the same created on every row, so the principal id keeps the
			// order total and paginated reads stable.
			"pullreq_reviewer_suggestion_principal_id ASC",
		).
		Limit(database.Limit(pagination.Size)).
		Offset(database.Offset(pagination.Page, pagination.Size))

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)
	var dst []pullReqReviewerSuggestion

	if err = db.SelectContext(ctx, &dst, sqlQuery, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list reviewer suggestions")
	}

	return mapPullReqReviewerSuggestions(dst), nil
}

func (s *pullReqReviewerSuggestionStore) Count(
	ctx context.Context,
	prID int64,
) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("pullreq_reviewer_suggestions").
		Where("pullreq_reviewer_suggestion_pullreq_id = ?", prID)

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.GetContext(ctx, &count, sqlQuery, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to count reviewer suggestions")
	}

	return count, nil
}

func (s *pullReqReviewerSuggestionStore) CreateMany(
	ctx context.Context,
	suggestions []*types.PullReqReviewerSuggestion,
) error {
	return s.insertMany(ctx, "create", suggestions, `
	ON CONFLICT
		(pullreq_reviewer_suggestion_pullreq_id, pullreq_reviewer_suggestion_principal_id)
	DO NOTHING
	`)
}

// UpsertMany inserts reviewer suggestions in batch and updates the detail fields of existing ones.
func (s *pullReqReviewerSuggestionStore) UpsertMany(
	ctx context.Context,
	suggestions []*types.PullReqReviewerSuggestion,
) error {
	return s.insertMany(ctx, "upsert", suggestions, `
	ON CONFLICT
		(pullreq_reviewer_suggestion_pullreq_id, pullreq_reviewer_suggestion_principal_id)
	DO UPDATE SET
		pullreq_reviewer_suggestion_reason = EXCLUDED.pullreq_reviewer_suggestion_reason
		,pullreq_reviewer_suggestion_kind = EXCLUDED.pullreq_reviewer_suggestion_kind
		,pullreq_reviewer_suggestion_rank = EXCLUDED.pullreq_reviewer_suggestion_rank
		,pullreq_reviewer_suggestion_user_group_id = EXCLUDED.pullreq_reviewer_suggestion_user_group_id
		,pullreq_reviewer_suggestion_commit_sha = EXCLUDED.pullreq_reviewer_suggestion_commit_sha
	`)
}

// insertMany runs the batch insert, operation names the caller ("create" or "upsert") in error messages.
func (s *pullReqReviewerSuggestionStore) insertMany(
	ctx context.Context,
	operation string,
	suggestions []*types.PullReqReviewerSuggestion,
	conflictSuffix string,
) error {
	if len(suggestions) == 0 {
		return nil
	}

	db := dbtx.GetAccessor(ctx, s.db)
	now := time.Now().UnixMilli()

	stmt := database.Builder.
		Insert("pullreq_reviewer_suggestions").
		Columns(pullReqReviewerSuggestionColumns)

	for _, suggestion := range suggestions {
		kind := suggestion.Kind
		if kind == "" {
			kind = enum.ReviewerSuggestionKindOptional
		}
		stmt = stmt.Values(
			suggestion.PullReqID,
			suggestion.CreatedBy,
			suggestion.PrincipalID,
			now,
			null.NewString(suggestion.Reason, suggestion.Reason != ""),
			string(kind),
			suggestion.Rank,
			suggestion.UserGroupID,
			null.NewString(suggestion.CommitSHA, suggestion.CommitSHA != ""),
		)
	}

	query, args, err := stmt.Suffix(conflictSuffix).ToSql()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	if _, err = db.ExecContext(ctx, query, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to %s reviewer suggestions", operation)
	}

	return nil
}

// Delete removes reviewer suggestion by pull request id and principal id.
func (s *pullReqReviewerSuggestionStore) Delete(ctx context.Context, prID, principalID int64) error {
	stmt := database.Builder.
		Delete("pullreq_reviewer_suggestions").
		Where(squirrel.Eq{
			"pullreq_reviewer_suggestion_pullreq_id":   prID,
			"pullreq_reviewer_suggestion_principal_id": principalID,
		})

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to delete reviewer suggestion")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to get number of deleted reviewer suggestions")
	}

	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

func mapPullReqReviewerSuggestions(dst []pullReqReviewerSuggestion) []*types.PullReqReviewerSuggestion {
	result := make([]*types.PullReqReviewerSuggestion, len(dst))
	for i, v := range dst {
		result[i] = mapPullReqReviewerSuggestion(&v)
	}
	return result
}

func mapPullReqReviewerSuggestion(v *pullReqReviewerSuggestion) *types.PullReqReviewerSuggestion {
	return &types.PullReqReviewerSuggestion{
		PullReqID:   v.PullReqID,
		CreatedBy:   v.CreatedBy,
		PrincipalID: v.PrincipalID,
		Created:     v.Created,
		Kind:        enum.ReviewerSuggestionKind(v.Kind),
		Reason:      v.Reason.String,
		Rank:        nullIntToPtr(v.Rank),
		UserGroupID: nullIntToPtr(v.UserGroupID),
		CommitSHA:   v.CommitSHA.String,
	}
}

func nullIntToPtr(v null.Int) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}
