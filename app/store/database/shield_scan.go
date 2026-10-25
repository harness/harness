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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.ShieldScanStore = (*ShieldScanStore)(nil)

// NewShieldScanStore returns a new ShieldScanStore.
func NewShieldScanStore(db *sqlx.DB) *ShieldScanStore {
	return &ShieldScanStore{db: db}
}

// ShieldScanStore implements store.ShieldScanStore backed by a relational database.
type ShieldScanStore struct {
	db *sqlx.DB
}

type shieldScan struct {
	ID            int64                  `db:"shield_scan_id"`
	RepoID        int64                  `db:"shield_scan_repo_id"`
	CommitSHA     string                 `db:"shield_scan_commit_sha"`
	Ref           string                 `db:"shield_scan_ref"`
	Trigger       enum.ShieldScanTrigger `db:"shield_scan_trigger"`
	Scanner       enum.ShieldScanner     `db:"shield_scan_scanner"`
	Status        enum.ShieldScanStatus  `db:"shield_scan_status"`
	Error         string                 `db:"shield_scan_error"`
	PackagesCount int                    `db:"shield_scan_packages_count"`
	FindingsCount int                    `db:"shield_scan_findings_count"`
	Started       int64                  `db:"shield_scan_started"`
	Finished      null.Int               `db:"shield_scan_finished"`
	Created       int64                  `db:"shield_scan_created"`
}

const (
	shieldScanColumns = `
		 shield_scan_id
		,shield_scan_repo_id
		,shield_scan_commit_sha
		,shield_scan_ref
		,shield_scan_trigger
		,shield_scan_scanner
		,shield_scan_status
		,shield_scan_error
		,shield_scan_packages_count
		,shield_scan_findings_count
		,shield_scan_started
		,shield_scan_finished
		,shield_scan_created`

	shieldScanSelectBase = `SELECT` + shieldScanColumns + `
		FROM shield_scans`
)

func (s *ShieldScanStore) Create(ctx context.Context, scan *types.ShieldScan) error {
	const sqlQuery = `
		INSERT INTO shield_scans (
			 shield_scan_repo_id
			,shield_scan_commit_sha
			,shield_scan_ref
			,shield_scan_trigger
			,shield_scan_scanner
			,shield_scan_status
			,shield_scan_error
			,shield_scan_packages_count
			,shield_scan_findings_count
			,shield_scan_started
			,shield_scan_finished
			,shield_scan_created
		) VALUES (
			 :shield_scan_repo_id
			,:shield_scan_commit_sha
			,:shield_scan_ref
			,:shield_scan_trigger
			,:shield_scan_scanner
			,:shield_scan_status
			,:shield_scan_error
			,:shield_scan_packages_count
			,:shield_scan_findings_count
			,:shield_scan_started
			,:shield_scan_finished
			,:shield_scan_created
		) RETURNING shield_scan_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalShieldScan(scan))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind shield scan object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&scan.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert shield scan")
	}

	return nil
}

func (s *ShieldScanStore) Update(ctx context.Context, scan *types.ShieldScan) error {
	const sqlQuery = `
		UPDATE shield_scans
		SET
			 shield_scan_commit_sha = :shield_scan_commit_sha
			,shield_scan_ref = :shield_scan_ref
			,shield_scan_status = :shield_scan_status
			,shield_scan_error = :shield_scan_error
			,shield_scan_packages_count = :shield_scan_packages_count
			,shield_scan_findings_count = :shield_scan_findings_count
			,shield_scan_started = :shield_scan_started
			,shield_scan_finished = :shield_scan_finished
		WHERE shield_scan_id = :shield_scan_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalShieldScan(scan))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind shield scan object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update shield scan")
	}

	return nil
}

func (s *ShieldScanStore) Find(ctx context.Context, id int64) (*types.ShieldScan, error) {
	const sqlQuery = shieldScanSelectBase + ` WHERE shield_scan_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &shieldScan{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find shield scan")
	}

	return mapShieldScan(dst), nil
}

func (s *ShieldScanStore) ListByRepo(ctx context.Context, repoID int64, page, size int) ([]*types.ShieldScan, error) {
	stmt := database.Builder.
		Select(shieldScanColumns).
		From("shield_scans").
		Where("shield_scan_repo_id = ?", repoID).
		OrderBy("shield_scan_created DESC", "shield_scan_id DESC").
		Limit(database.Limit(size)).
		Offset(database.Offset(page, size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert shield scan list query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*shieldScan{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing shield scan list query")
	}

	out := make([]*types.ShieldScan, len(dst))
	for i, d := range dst {
		out[i] = mapShieldScan(d)
	}

	return out, nil
}

func mapInternalShieldScan(s *types.ShieldScan) *shieldScan {
	return &shieldScan{
		ID:            s.ID,
		RepoID:        s.RepoID,
		CommitSHA:     s.CommitSHA,
		Ref:           s.Ref,
		Trigger:       s.Trigger,
		Scanner:       s.Scanner,
		Status:        s.Status,
		Error:         s.Error,
		PackagesCount: s.PackagesCount,
		FindingsCount: s.FindingsCount,
		Started:       s.Started,
		Finished:      null.IntFromPtr(s.Finished),
		Created:       s.Created,
	}
}

func mapShieldScan(s *shieldScan) *types.ShieldScan {
	return &types.ShieldScan{
		ID:            s.ID,
		RepoID:        s.RepoID,
		CommitSHA:     s.CommitSHA,
		Ref:           s.Ref,
		Trigger:       s.Trigger,
		Scanner:       s.Scanner,
		Status:        s.Status,
		Error:         s.Error,
		PackagesCount: s.PackagesCount,
		FindingsCount: s.FindingsCount,
		Started:       s.Started,
		Finished:      s.Finished.Ptr(),
		Created:       s.Created,
	}
}
