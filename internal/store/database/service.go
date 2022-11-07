// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
)

var _ store.ServiceStore = (*ServiceStore)(nil)

// service is a DB representation of a service principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type service struct {
	types.Service
	UIDUnique string `db:"principal_uidUnique"`
}

// NewServiceStore returns a new ServiceStore.
func NewServiceStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) *ServiceStore {
	return &ServiceStore{
		db:                db,
		uidTransformation: uidTransformation,
	}
}

// ServiceStore implements a ServiceStore backed by a relational
// database.
type ServiceStore struct {
	db                *sqlx.DB
	uidTransformation store.PrincipalUIDTransformation
}

// Find finds the service by id.
func (s *ServiceStore) Find(ctx context.Context, id int64) (*types.Service, error) {
	dst := new(service)
	if err := s.db.GetContext(ctx, dst, serviceSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}
	return s.mapDBService(dst), nil
}

// FindUID finds the service by uid.
func (s *ServiceStore) FindUID(ctx context.Context, uid string) (*types.Service, error) {
	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(enum.PrincipalTypeService, uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, store.ErrResourceNotFound
	}

	dst := new(service)
	if err = s.db.GetContext(ctx, dst, serviceSelectUIDUnique, uidUnique); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}
	return s.mapDBService(dst), nil
}

// Create saves the service.
func (s *ServiceStore) Create(ctx context.Context, svc *types.Service) error {
	dbSVC, err := s.mapToDBservice(svc)
	if err != nil {
		return fmt.Errorf("failed to map db service: %w", err)
	}

	query, arg, err := s.db.BindNamed(serviceInsert, dbSVC)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&svc.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the service.
func (s *ServiceStore) Update(ctx context.Context, svc *types.Service) error {
	dbSVC, err := s.mapToDBservice(svc)
	if err != nil {
		return fmt.Errorf("failed to map db service: %w", err)
	}

	query, arg, err := s.db.BindNamed(serviceUpdate, dbSVC)
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
	dst := []*service{}

	err := s.db.SelectContext(ctx, &dst, serviceSelect)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed executing default list query")
	}
	return s.mapDBServices(dst), nil
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

func (s *ServiceStore) mapDBService(dbSvc *service) *types.Service {
	return &dbSvc.Service
}

func (s *ServiceStore) mapDBServices(dbSVCs []*service) []*types.Service {
	res := make([]*types.Service, len(dbSVCs))
	for i := range dbSVCs {
		res[i] = s.mapDBService(dbSVCs[i])
	}
	return res
}

func (s *ServiceStore) mapToDBservice(svc *types.Service) (*service, error) {
	// service comes from outside.
	if svc == nil {
		return nil, fmt.Errorf("service is nil")
	}

	uidUnique, err := s.uidTransformation(enum.PrincipalTypeService, svc.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform service UID: %w", err)
	}
	dbService := &service{
		Service:   *svc,
		UIDUnique: uidUnique,
	}

	return dbService, nil
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
,principal_uidUnique
,principal_email
,principal_displayName
,principal_blocked
,principal_salt
,principal_created
,principal_updated
FROM principals
`

const serviceSelect = serviceBase + `
WHERE principal_type = "service"
ORDER BY principal_uid ASC
`

const serviceSelectID = serviceBase + `
WHERE principal_type = "service" AND principal_id = $1
`

const serviceSelectUIDUnique = serviceBase + `
WHERE principal_type = "service" AND principal_uidUnique = $1
`

const serviceDelete = `
DELETE FROM principals
WHERE principal_type = "service" AND principal_id = $1
`

const serviceInsert = `
INSERT INTO principals (
principal_type
,principal_uid
,principal_uidUnique
,principal_email
,principal_displayName
,principal_admin
,principal_blocked
,principal_salt
,principal_created
,principal_updated
) values (
 "service"
,:principal_uid
,:principal_uidUnique
,:principal_email
,:principal_displayName
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
principal_email     	  = :principal_email
,principal_displayName    = :principal_displayName
,principal_admin          = :principal_admin
,principal_blocked        = :principal_blocked
,principal_updated        = :principal_updated
WHERE principal_type = "service" AND principal_id = :principal_id
`
