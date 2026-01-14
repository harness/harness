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
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ store.PullReqReviewerStore = (*PullReqReviewerStore)(nil)

const maxPullRequestReviewers = 100

// NewPullReqReviewerStore returns a new PullReqReviewerStore.
func NewPullReqReviewerStore(db *sqlx.DB,
	pCache store.PrincipalInfoCache) *PullReqReviewerStore {
	return &PullReqReviewerStore{
		db:     db,
		pCache: pCache,
	}
}

// PullReqReviewerStore implements store.PullReqReviewerStore backed by a relational database.
type PullReqReviewerStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
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
		,pullreq_reviewer_sha`

	pullreqReviewerSelectBase = `
	SELECT` + pullreqReviewerColumns + `
	FROM pullreq_reviewers`
)

// Find finds the pull request reviewer by pull request id and principal id.
func (s *PullReqReviewerStore) Find(ctx context.Context, prID, principalID int64) (*types.PullReqReviewer, error) {
	const sqlQuery = pullreqReviewerSelectBase + `
	WHERE pullreq_reviewer_pullreq_id = $1 AND pullreq_reviewer_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReqReviewer{}
	if err := db.GetContext(ctx, dst, sqlQuery, prID, principalID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request reviewer")
	}

	return s.mapPullReqReviewer(ctx, dst), nil
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
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request reviewer object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert pull request reviewer")
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
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request activity object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update pull request activity")
	}

	v.Updated = dbv.Updated

	return nil
}

// Delete deletes the pull request reviewer.
func (s *PullReqReviewerStore) Delete(ctx context.Context, prID, reviewerID int64) error {
	const sqlQuery = `
	DELETE from pullreq_reviewers
	WHERE pullreq_reviewer_pullreq_id = $1 AND
	      pullreq_reviewer_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, prID, reviewerID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "delete reviewer query failed")
	}
	return nil
}

// List returns a list of pull reviewers for a pull request.
func (s *PullReqReviewerStore) List(ctx context.Context, prID int64) ([]*types.PullReqReviewer, error) {
	stmt := database.Builder.
		Select(pullreqReviewerColumns).
		From("pullreq_reviewers").
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
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing pull request reviewer list query")
	}

	result, err := s.mapSlicePullReqReviewer(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
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
	}
	return m
}

func (s *PullReqReviewerStore) mapPullReqReviewer(ctx context.Context, v *pullReqReviewer) *types.PullReqReviewer {
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
	}

	addedBy, err := s.pCache.Get(ctx, v.CreatedBy)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to load PR reviewer addedBy")
	}
	if addedBy != nil {
		m.AddedBy = *addedBy
	}

	reviewer, err := s.pCache.Get(ctx, v.PrincipalID)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to load PR reviewer principal")
	}
	if reviewer != nil {
		m.Reviewer = *reviewer
	}

	return m
}

func (s *PullReqReviewerStore) mapSlicePullReqReviewer(ctx context.Context,
	reviewers []*pullReqReviewer) ([]*types.PullReqReviewer, error) {
	// collect all principal IDs
	ids := make([]int64, 0, 2*len(reviewers))
	for _, v := range reviewers {
		ids = append(ids, v.CreatedBy)
		ids = append(ids, v.PrincipalID)
	}

	// pull principal infos from cache
	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load PR principal infos: %w", err)
	}

	// attach the principal infos back to the slice items
	m := make([]*types.PullReqReviewer, len(reviewers))
	for i, v := range reviewers {
		m[i] = mapPullReqReviewer(v)
		if addedBy, ok := infoMap[v.CreatedBy]; ok {
			m[i].AddedBy = *addedBy
		}
		if reviewer, ok := infoMap[v.PrincipalID]; ok {
			m[i].Reviewer = *reviewer
		}
	}

	return m, nil
}
