// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"database/sql"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.ServiceStore = (*ServiceStore)(nil)

// NewServiceStore returns a new ServiceStore.
func NewServiceStore(db *sqlx.DB) *ServiceStore {
	return &ServiceStore{db}
}

// ServiceStore implements a ServiceStore backed by a relational
// database.
type ServiceStore struct {
	db *sqlx.DB
}

// Find finds the service by id.
func (s *ServiceStore) Find(ctx context.Context, id int64) (*types.Service, error) {
	dst := new(types.Service)
	if err := s.db.GetContext(ctx, dst, serviceSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}
	return dst, nil
}

// FindUID finds the service by uid.
func (s *ServiceStore) FindUID(ctx context.Context, uid string) (*types.Service, error) {
	dst := new(types.Service)
	if err := s.db.GetContext(ctx, dst, serviceSelectUID, uid); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}
	return dst, nil
}

// Create saves the service.
func (s *ServiceStore) Create(ctx context.Context, sa *types.Service) error {
	query, arg, err := s.db.BindNamed(serviceInsert, sa)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&sa.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the service.
func (s *ServiceStore) Update(ctx context.Context, sa *types.Service) error {
	query, arg, err := s.db.BindNamed(serviceUpdate, sa)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the service.
func (s *ServiceStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)
	// delete the service
	if _, err = tx.ExecContext(ctx, serviceDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}
	return tx.Commit()
}

// List returns a list of service for a specific parent.
func (s *ServiceStore) List(ctx context.Context) ([]*types.Service, error) {
	dst := []*types.Service{}

	err := s.db.SelectContext(ctx, &dst, serviceSelect)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed executing default list query")
	}
	return dst, nil
}

// Count returns a count of service for a specific parent.
func (s *ServiceStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, serviceCount).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

const serviceCount = `
SELECT count(*)
FROM principals
WHERE principal_type = "service"
`

const serviceBase = `
SELECT
principal_id
,principal_uid
,principal_name
,principal_blocked
,principal_salt
,principal_created
,principal_updated
FROM principals
`

const serviceSelect = serviceBase + `
WHERE principal_type = "service"
ORDER BY principal_name ASC
`

const serviceSelectID = serviceBase + `
WHERE principal_type = "service" AND principal_id = $1
`

const serviceSelectUID = serviceBase + `
WHERE principal_type = "service" AND principal_uid = $1
`

const serviceDelete = `
DELETE FROM principals
WHERE principal_type = "service" AND principal_id = $1
`

const serviceInsert = `
INSERT INTO principals (
principal_type
,principal_uid
,principal_name
,principal_admin
,principal_blocked
,principal_salt
,principal_created
,principal_updated
) values (
 "service"
,:principal_uid
,:principal_name
,:principal_admin
,:principal_blocked
,:principal_salt
,:principal_created
,:principal_updated
) RETURNING principal_id
`

const serviceUpdate = `
UPDATE principals
SET
 principal_name            = :principal_name
,principal_admin        = :principal_admin
,principal_blocked        = :principal_blocked
,principal_updated        = :principal_updated
WHERE principal_type = "service" AND principal_id = :principal_id
`
