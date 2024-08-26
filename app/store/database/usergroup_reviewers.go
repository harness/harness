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

	"github.com/jmoiron/sqlx"
)

var _ store.PullReqReviewerStore = (*PullReqReviewerStore)(nil)

func NewUsergroupReviewerStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
	userGroupStore store.UserGroupStore,
) *UsergroupReviewerStore {
	return &UsergroupReviewerStore{
		db:             db,
		pInfoCache:     pCache,
		userGroupStore: userGroupStore,
	}
}

// UsergroupReviewerStore implements store.UsergroupReviewerStore backed by a relational database.
type UsergroupReviewerStore struct {
	db             *sqlx.DB
	pInfoCache     store.PrincipalInfoCache
	userGroupStore store.UserGroupStore
}

type usergroupReviewer struct {
	PullReqID   int64 `db:"usergroup_reviewer_pullreq_id"`
	UserGroupID int64 `db:"usergroup_reviewer_usergroup_id"`
	CreatedBy   int64 `db:"usergroup_reviewer_created_by"`
	Created     int64 `db:"usergroup_reviewer_created"`
	Updated     int64 `db:"usergroup_reviewer_updated"`
	RepoID      int64 `db:"usergroup_reviewer_repo_id"`
}

const (
	pullreqUserGroupReviewerColumns = `
		 usergroup_reviewer_pullreq_id
		,usergroup_reviewer_usergroup_id
		,usergroup_reviewer_created_by
		,usergroup_reviewer_created
		,usergroup_reviewer_updated
		,usergroup_reviewer_repo_id`

	pullreqUserGroupReviewerSelectBase = `
	SELECT` + pullreqUserGroupReviewerColumns + `
	FROM usergroup_reviewers`
)

// Create creates a new pull request usergroup reviewer.
func (s *UsergroupReviewerStore) Create(ctx context.Context, v *types.UserGroupReviewer) error {
	const sqlQuery = `
		INSERT INTO usergroup_reviewers (
			usergroup_reviewer_pullreq_id,
			usergroup_reviewer_usergroup_id,
			usergroup_reviewer_created_by,
			usergroup_reviewer_created,
			usergroup_reviewer_updated,
			usergroup_reviewer_repo_id
		) VALUES (
			:usergroup_reviewer_pullreq_id,
			:usergroup_reviewer_usergroup_id,
			:usergroup_reviewer_created_by,
			:usergroup_reviewer_created,
			:usergroup_reviewer_updated,
			:usergroup_reviewer_repo_id
			)`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReqUserGroupReviewer(v))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pull request usergroup reviewer object")
	}

	if _, err := db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert pull request usergroup reviewer")
	}
	return nil
}

// Delete deletes a pull request usergroup reviewer.
func (s *UsergroupReviewerStore) Delete(ctx context.Context, prID, userGroupReviewerID int64) error {
	const sqlQuery = `
		DELETE FROM usergroup_reviewers
		WHERE usergroup_reviewer_pullreq_id = $1 AND usergroup_reviewer_usergroup_id = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, prID, userGroupReviewerID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete pull request usergroup reviewer")
	}
	return nil
}

// List returns a list of pull request usergroup reviewers.
func (s *UsergroupReviewerStore) List(ctx context.Context, prID int64) ([]*types.UserGroupReviewer, error) {
	const sqlQuery = pullreqUserGroupReviewerSelectBase + `
	WHERE usergroup_reviewer_pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*usergroupReviewer
	if err := db.SelectContext(ctx, &dst, sqlQuery, prID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list pull request usergroup reviewers")
	}

	result, err := s.mapSlicePullReqUserGroupReviewer(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Find returns a pull request usergroup reviewer by userGroupReviewerID.
func (s *UsergroupReviewerStore) Find(
	ctx context.Context,
	prID,
	userGroupReviewerID int64,
) (*types.UserGroupReviewer, error) {
	const sqlQuery = pullreqUserGroupReviewerSelectBase + `
	WHERE usergroup_reviewer_pullreq_id = $1 AND usergroup_reviewer_usergroup_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)
	dst := &usergroupReviewer{}
	if err := db.GetContext(ctx, dst, sqlQuery, prID, userGroupReviewerID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find pull request usergroup reviewer")
	}

	return mapPullReqUserGroupReviewer(dst), nil
}

func mapInternalPullReqUserGroupReviewer(v *types.UserGroupReviewer) *usergroupReviewer {
	m := &usergroupReviewer{
		PullReqID:   v.PullReqID,
		UserGroupID: v.UserGroupID,
		CreatedBy:   v.CreatedBy,
		Created:     v.Created,
		Updated:     v.Updated,
		RepoID:      v.RepoID,
	}
	return m
}

func mapPullReqUserGroupReviewer(v *usergroupReviewer) *types.UserGroupReviewer {
	m := &types.UserGroupReviewer{
		PullReqID:   v.PullReqID,
		UserGroupID: v.UserGroupID,
		CreatedBy:   v.CreatedBy,
		Created:     v.Created,
		Updated:     v.Updated,
		RepoID:      v.RepoID,
	}
	return m
}

func (s *UsergroupReviewerStore) mapSlicePullReqUserGroupReviewer(
	ctx context.Context,
	userGroupReviewers []*usergroupReviewer,
) ([]*types.UserGroupReviewer, error) {
	result := make([]*types.UserGroupReviewer, 0, len(userGroupReviewers))
	var addedByIDs []int64
	var userGroupIDs []int64
	for _, v := range userGroupReviewers {
		addedByIDs = append(addedByIDs, v.CreatedBy)
		userGroupIDs = append(userGroupIDs, v.UserGroupID)
	}

	// pull all the usergroups info
	userGroupsMap, err := s.userGroupStore.Map(ctx, userGroupIDs)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to load PR usergroups")
	}

	// pull principal infos from cache
	infoMap, err := s.pInfoCache.Map(ctx, addedByIDs)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to load PR principal infos")
	}

	for _, v := range userGroupReviewers {
		pullReqUsergroupReviewer := mapPullReqUserGroupReviewer(v)
		pullReqUsergroupReviewer.UserGroup = *userGroupsMap[v.UserGroupID].ToUserGroupInfo()
		pullReqUsergroupReviewer.AddedBy = *infoMap[v.CreatedBy]

		result = append(result, pullReqUsergroupReviewer)
	}
	return result, nil
}
