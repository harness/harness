// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"database/sql"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.ServiceAccountStore = (*ServiceAccountStore)(nil)

// NewServiceAccountStore returns a new ServiceAccountStore.
func NewServiceAccountStore(db *sqlx.DB) *ServiceAccountStore {
	return &ServiceAccountStore{db}
}

// ServiceAccountStore implements a ServiceAccountStore backed by a relational
// database.
type ServiceAccountStore struct {
	db *sqlx.DB
}

// Find finds the service account by id.
func (s *ServiceAccountStore) Find(ctx context.Context, id int64) (*types.ServiceAccount, error) {
	dst := new(types.ServiceAccount)
	if err := s.db.GetContext(ctx, dst, serviceAccountSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}
	return dst, nil
}

// FindUID finds the service account by uid.
func (s *ServiceAccountStore) FindUID(ctx context.Context, uid string) (*types.ServiceAccount, error) {
	dst := new(types.ServiceAccount)
	if err := s.db.GetContext(ctx, dst, serviceAccountSelectUID, uid); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}
	return dst, nil
}

// Create saves the service account.
func (s *ServiceAccountStore) Create(ctx context.Context, sa *types.ServiceAccount) error {
	query, arg, err := s.db.BindNamed(serviceAccountInsert, sa)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service account object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&sa.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the service account details.
func (s *ServiceAccountStore) Update(ctx context.Context, sa *types.ServiceAccount) error {
	query, arg, err := s.db.BindNamed(serviceAccountUpdate, sa)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service account object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the service account.
func (s *ServiceAccountStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)
	// delete the service account
	if _, err = tx.ExecContext(ctx, serviceAccountDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}
	return tx.Commit()
}

// List returns a list of service accounts for a specific parent.
func (s *ServiceAccountStore) List(ctx context.Context, parentType enum.ParentResourceType,
	parentID int64) ([]*types.ServiceAccount, error) {
	dst := []*types.ServiceAccount{}

	err := s.db.SelectContext(ctx, &dst, serviceAccountSelectByParentTypeAndID, parentType, parentID)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed executing default list query")
	}
	return dst, nil
}

// Count returns a count of service accounts for a specific parent.
func (s *ServiceAccountStore) Count(ctx context.Context,
	parentType enum.ParentResourceType, parentID int64) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, serviceAccountCountByParentTypeAndID, parentType, parentID).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

const serviceAccountCountByParentTypeAndID = `
SELECT count(*)
FROM principals
WHERE principal_type = "serviceaccount" and principal_sa_parentType = $1 and principal_sa_parentId = $2
`

const serviceAccountBase = `
SELECT
principal_id
,principal_uid
,principal_name
,principal_blocked
,principal_salt
,principal_created
,principal_updated
,principal_sa_parentType
,principal_sa_parentId
FROM principals
`

const serviceAccountSelectByParentTypeAndID = serviceAccountBase + `
WHERE principal_type = "serviceaccount" AND principal_sa_parentType = $1 AND principal_sa_parentId = $2
ORDER BY principal_name ASC
`

const serviceAccountSelectID = serviceAccountBase + `
WHERE principal_type = "serviceaccount" AND principal_id = $1
`

const serviceAccountSelectUID = serviceAccountBase + `
WHERE principal_type = "serviceaccount" AND principal_uid = $1
`

const serviceAccountDelete = `
DELETE FROM principals
WHERE principal_type = "serviceaccount" AND principal_id = $1
`

const serviceAccountInsert = `
INSERT INTO principals (
principal_type
,principal_uid
,principal_name
,principal_admin
,principal_blocked
,principal_salt
,principal_created
,principal_updated
,principal_sa_parentType
,principal_sa_parentId
) values (
 "serviceaccount"
,:principal_uid
,:principal_name
,false
,:principal_blocked
,:principal_salt
,:principal_created
,:principal_updated
,:principal_sa_parentType
,:principal_sa_parentId
) RETURNING principal_id
`

const serviceAccountUpdate = `
UPDATE principals
SET
 principal_name            = :principal_name
,:principal_blocked        = :principal_blocked
,:principal_salt           = :principal_salt
,:principal_updated        = :principal_updated
WHERE principal_type = "serviceaccount" AND principal_id = :principal_id
`
