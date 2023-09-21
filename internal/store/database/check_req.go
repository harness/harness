// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.ReqCheckStore = (*ReqCheckStore)(nil)

// NewReqCheckStore returns a new CheckStore.
func NewReqCheckStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
) *ReqCheckStore {
	return &ReqCheckStore{
		db:     db,
		pCache: pCache,
	}
}

// ReqCheckStore implements store.CheckStore backed by a relational database.
type ReqCheckStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

const (
	reqCheckColumns = `
		 reqcheck_id
		,reqcheck_created_by
		,reqcheck_created
		,reqcheck_repo_id
		,reqcheck_branch_pattern
		,reqcheck_check_uid`
)

// reqCheck is used to fetch required status check data from the database.
// The object should be later re-packed into a different struct to return it as an API response.
type reqCheck struct {
	ID            int64  `db:"reqcheck_id"`
	CreatedBy     int64  `db:"reqcheck_created_by"`
	Created       int64  `db:"reqcheck_created"`
	RepoID        int64  `db:"reqcheck_repo_id"`
	BranchPattern string `db:"reqcheck_branch_pattern"`
	CheckUID      string `db:"reqcheck_check_uid"`
}

// Create creates new required status check.
func (s *ReqCheckStore) Create(ctx context.Context, reqCheck *types.ReqCheck) error {
	const sqlQuery = `
	INSERT INTO reqchecks (
		 reqcheck_created_by
		,reqcheck_created
		,reqcheck_repo_id
		,reqcheck_branch_pattern
		,reqcheck_check_uid
	) VALUES (
		 :reqcheck_created_by
		,:reqcheck_created
		,:reqcheck_repo_id
		,:reqcheck_branch_pattern
		,:reqcheck_check_uid
	)
	RETURNING reqcheck_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalReqCheck(reqCheck))
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind required status check object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&reqCheck.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// List returns a list of required status checks for a repo.
func (s *ReqCheckStore) List(ctx context.Context, repoID int64) ([]*types.ReqCheck, error) {
	stmt := database.Builder.
		Select(reqCheckColumns).
		From("reqchecks").
		Where("reqcheck_repo_id = ?", repoID).
		OrderBy("reqcheck_check_uid")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	dst := make([]*reqCheck, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to execute list required status checks query")
	}

	result, err := s.mapSliceReqCheck(ctx, dst)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete removes a required status checks for a repo.
func (s *ReqCheckStore) Delete(ctx context.Context, repoID, reqCheckID int64) error {
	stmt := database.Builder.
		Delete("reqchecks").
		Where("reqcheck_repo_id = ?", repoID).
		Where("reqcheck_id = ?", reqCheckID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to execute delete required status check query")
	}

	return nil
}

func mapReqCheck(req *reqCheck) *types.ReqCheck {
	return &types.ReqCheck{
		ID:            req.ID,
		CreatedBy:     req.CreatedBy,
		Created:       req.Created,
		RepoID:        req.RepoID,
		BranchPattern: req.BranchPattern,
		CheckUID:      req.CheckUID,
		AddedBy:       types.PrincipalInfo{},
	}
}

func mapInternalReqCheck(req *types.ReqCheck) *reqCheck {
	m := &reqCheck{
		ID:            req.ID,
		CreatedBy:     req.CreatedBy,
		Created:       req.Created,
		RepoID:        req.RepoID,
		BranchPattern: req.BranchPattern,
		CheckUID:      req.CheckUID,
	}

	return m
}

func (s *ReqCheckStore) mapSliceReqCheck(ctx context.Context, reqChecks []*reqCheck) ([]*types.ReqCheck, error) {
	// collect all principal IDs
	ids := make([]int64, len(reqChecks))
	for i, req := range reqChecks {
		ids[i] = req.CreatedBy
	}

	// pull principal infos from cache
	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load required status check principal infos: %w", err)
	}

	// attach the principal infos back to the slice items
	m := make([]*types.ReqCheck, len(reqChecks))
	for i, req := range reqChecks {
		m[i] = mapReqCheck(req)
		if author, ok := infoMap[req.CreatedBy]; ok {
			m[i].AddedBy = *author
		}
	}

	return m, nil
}
