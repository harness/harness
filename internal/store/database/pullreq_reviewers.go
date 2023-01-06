// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqReviewerStore = (*PullReqReviewerStore)(nil)

const maxPullRequestReviewers = 100

// NewPullReqReviewerStore returns a new PullReqReviewerStore.
func NewPullReqReviewerStore(db *sqlx.DB) *PullReqReviewerStore {
	return &PullReqReviewerStore{
		db: db,
	}
}

// PullReqReviewerStore implements store.PullReqReviewerStore backed by a relational database.
type PullReqReviewerStore struct {
	db *sqlx.DB
}

// pullReqReviewer is used to fetch pull request reviewer data from the database.
type pullReqReviewer struct {
	PullReqID   int64 `db:"pullreq_reviewer_pullreq_id"`
	PrincipalID int64 `db:"pullreq_reviewer_principal_id"`
	CreatedBy   int64 `db:"pullreq_reviewer_created_by"`
	Created     int64 `db:"pullreq_reviewer_created"`
	Updated     int64 `db:"pullreq_reviewer_updated"`

	RepoID         int64                    `db:"pullreq_reviewer_repo_id"`
	Type           enum.PullReqReviewerType `db:"pullreq_reviewer_type"`
	LatestReviewID null.Int                 `db:"pullreq_reviewer_latest_review_id"`

	ReviewDecision enum.PullReqReviewDecision `db:"pullreq_reviewer_review_decision"`
	SHA            string                     `db:"pullreq_reviewer_sha"`

	ReviewerUID   string `db:"reviewer_uid"`
	ReviewerName  string `db:"reviewer_name"`
	ReviewerEmail string `db:"reviewer_email"`
	AddedByUID    string `db:"added_by_uid"`
	AddedByName   string `db:"added_by_name"`
	AddedByEmail  string `db:"added_by_email"`
}

const (
	pullreqReviewerColumns = `
		 pullreq_reviewer_pullreq_id
		,pullreq_reviewer_principal_id
		,pullreq_reviewer_created_by
		,pullreq_reviewer_created
		,pullreq_reviewer_updated
		,pullreq_reviewer_repo_id
		,pullreq_reviewer_type
		,pullreq_reviewer_latest_review_id
		,pullreq_reviewer_review_decision
		,pullreq_reviewer_sha
		,reviewer.principal_uid as "reviewer_uid"
		,reviewer.principal_display_name as "reviewer_name"
		,reviewer.principal_email as "reviewer_email"
		,added_by.principal_uid as "added_by_uid"
		,added_by.principal_display_name as "added_by_name"
		,added_by.principal_email as "added_by_email"`

	pullreqReviewerSelectBase = `
	SELECT` + pullreqReviewerColumns + `
	FROM pullreq_reviewers
	INNER JOIN principals reviewer on reviewer.principal_id = pullreq_reviewer_principal_id
	INNER JOIN principals added_by on added_by.principal_id = pullreq_reviewer_created_by`
)

// Find finds the pull request reviewer by pull request id and principal id.
func (s *PullReqReviewerStore) Find(ctx context.Context, prID, principalID int64) (*types.PullReqReviewer, error) {
	const sqlQuery = pullreqReviewerSelectBase + `
	WHERE pullreq_reviewer_pullreq_id = $1 AND pullreq_reviewer_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReqReviewer{}
	if err := db.GetContext(ctx, dst, sqlQuery, prID, principalID); err != nil {
		return nil, processSQLErrorf(err, "Failed to find pull request reviewer")
	}

	return mapPullReqReviewer(dst), nil
}

// Create creates a new pull request reviewer.
func (s *PullReqReviewerStore) Create(ctx context.Context, v *types.PullReqReviewer) error {
	const sqlQuery = `
	INSERT INTO pullreq_reviewers (
		 pullreq_reviewer_pullreq_id
		,pullreq_reviewer_principal_id
		,pullreq_reviewer_created_by
		,pullreq_reviewer_created
		,pullreq_reviewer_updated
		,pullreq_reviewer_repo_id
		,pullreq_reviewer_type
		,pullreq_reviewer_latest_review_id
		,pullreq_reviewer_review_decision
		,pullreq_reviewer_sha
	) values (
		 :pullreq_reviewer_pullreq_id
		,:pullreq_reviewer_principal_id
		,:pullreq_reviewer_created_by
		,:pullreq_reviewer_created
		,:pullreq_reviewer_updated
		,:pullreq_reviewer_repo_id
		,:pullreq_reviewer_type
		,:pullreq_reviewer_latest_review_id
		,:pullreq_reviewer_review_decision
		,:pullreq_reviewer_sha
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReqReviewer(v))
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request reviewer object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Failed to insert pull request reviewer")
	}

	return nil
}

// Update updates the pull request reviewer.
func (s *PullReqReviewerStore) Update(ctx context.Context, v *types.PullReqReviewer) error {
	const sqlQuery = `
	UPDATE pullreq_reviewers
	SET
		 pullreq_reviewer_updated = :pullreq_reviewer_updated
		,pullreq_reviewer_latest_review_id = :pullreq_reviewer_latest_review_id
		,pullreq_reviewer_review_decision = :pullreq_reviewer_review_decision
		,pullreq_reviewer_sha = :pullreq_reviewer_sha
	WHERE pullreq_reviewer_pullreq_id = :pullreq_reviewer_pullreq_id AND
	      pullreq_reviewer_principal_id = :pullreq_reviewer_principal_id`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbv := mapInternalPullReqReviewer(v)
	dbv.Updated = updatedAt.UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbv)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request activity object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to update pull request activity")
	}

	v.Updated = dbv.Updated

	return nil
}

// List returns a list of pull reviewers for a pull request.
func (s *PullReqReviewerStore) List(ctx context.Context, prID int64) ([]*types.PullReqReviewer, error) {
	stmt := builder.
		Select(pullreqReviewerColumns).
		From("pullreq_reviewers").
		InnerJoin("principals reviewer on reviewer.principal_id = pullreq_reviewer_principal_id").
		LeftJoin("principals added_by on added_by.principal_id = pullreq_reviewer_created_by").
		Where("pullreq_reviewer_pullreq_id = ?", prID).
		OrderBy("pullreq_reviewer_created asc").
		Limit(maxPullRequestReviewers) // memory safety limit

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request reviewer list query to sql")
	}

	dst := make([]*pullReqReviewer, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing pull request reviewer list query")
	}

	return mapSlicePullReqReviewer(dst), nil
}

func mapPullReqReviewer(v *pullReqReviewer) *types.PullReqReviewer {
	m := &types.PullReqReviewer{
		PullReqID:      v.PullReqID,
		PrincipalID:    v.PrincipalID,
		CreatedBy:      v.CreatedBy,
		Created:        v.Created,
		Updated:        v.Updated,
		RepoID:         v.RepoID,
		Type:           v.Type,
		LatestReviewID: v.LatestReviewID.Ptr(),
		ReviewDecision: v.ReviewDecision,
		SHA:            v.SHA,
		Reviewer: types.PrincipalInfo{
			ID:          v.PrincipalID,
			UID:         v.ReviewerUID,
			DisplayName: v.ReviewerName,
			Email:       v.ReviewerEmail,
		},
		AddedBy: types.PrincipalInfo{
			ID:          v.CreatedBy,
			UID:         v.AddedByUID,
			DisplayName: v.AddedByName,
			Email:       v.AddedByEmail,
		},
	}
	return m
}

func mapInternalPullReqReviewer(v *types.PullReqReviewer) *pullReqReviewer {
	m := &pullReqReviewer{
		PullReqID:      v.PullReqID,
		PrincipalID:    v.PrincipalID,
		CreatedBy:      v.CreatedBy,
		Created:        v.Created,
		Updated:        v.Updated,
		RepoID:         v.RepoID,
		Type:           v.Type,
		LatestReviewID: null.IntFromPtr(v.LatestReviewID),
		ReviewDecision: v.ReviewDecision,
		SHA:            v.SHA,
		ReviewerUID:    "",
		ReviewerName:   "",
		ReviewerEmail:  "",
		AddedByUID:     "",
		AddedByName:    "",
		AddedByEmail:   "",
	}
	return m
}

func mapSlicePullReqReviewer(a []*pullReqReviewer) []*types.PullReqReviewer {
	m := make([]*types.PullReqReviewer, len(a))
	for i, act := range a {
		m[i] = mapPullReqReviewer(act)
	}
	return m
}
