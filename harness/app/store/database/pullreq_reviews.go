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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.PullReqReviewStore = (*PullReqReviewStore)(nil)

// NewPullReqReviewStore returns a new PullReqReviewStore.
func NewPullReqReviewStore(db *sqlx.DB) *PullReqReviewStore {
	return &PullReqReviewStore{
		db: db,
	}
}

// PullReqReviewStore implements store.PullReqReviewStore backed by a relational database.
type PullReqReviewStore struct {
	db *sqlx.DB
}

// pullReqReview is used to fetch pull request review data from the database.
type pullReqReview struct {
	ID int64 `db:"pullreq_review_id"`

	CreatedBy int64 `db:"pullreq_review_created_by"`
	Created   int64 `db:"pullreq_review_created"`
	Updated   int64 `db:"pullreq_review_updated"`

	PullReqID int64 `db:"pullreq_review_pullreq_id"`

	Decision enum.PullReqReviewDecision `db:"pullreq_review_decision"`
	SHA      string                     `db:"pullreq_review_sha"`
}

const (
	pullreqReviewColumns = `
		 pullreq_review_id
		,pullreq_review_created_by
		,pullreq_review_created
		,pullreq_review_updated
		,pullreq_review_pullreq_id
		,pullreq_review_decision
		,pullreq_review_sha`

	pullreqReviewSelectBase = `
	SELECT` + pullreqReviewColumns + `
	FROM pullreq_reviews`
)

// Find finds the pull request activity by id.
func (s *PullReqReviewStore) Find(ctx context.Context, id int64) (*types.PullReqReview, error) {
	const sqlQuery = pullreqReviewSelectBase + `
	WHERE pullreq_review_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReqReview{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request activity")
	}

	return mapPullReqReview(dst), nil
}

// Create creates a new pull request.
func (s *PullReqReviewStore) Create(ctx context.Context, v *types.PullReqReview) error {
	const sqlQuery = `
	INSERT INTO pullreq_reviews (
		 pullreq_review_created_by
		,pullreq_review_created
		,pullreq_review_updated
		,pullreq_review_pullreq_id
		,pullreq_review_decision
		,pullreq_review_sha
	) values (
		 :pullreq_review_created_by
		,:pullreq_review_created
		,:pullreq_review_updated
		,:pullreq_review_pullreq_id
		,:pullreq_review_decision
		,:pullreq_review_sha
	) RETURNING pullreq_review_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReqReview(v))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request review object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&v.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert pull request review")
	}

	return nil
}

func mapPullReqReview(v *pullReqReview) *types.PullReqReview {
	return (*types.PullReqReview)(v) // the two types are identical, except for the tags
}

func mapInternalPullReqReview(v *types.PullReqReview) *pullReqReview {
	return (*pullReqReview)(v) // the two types are identical, except for the tags
}
