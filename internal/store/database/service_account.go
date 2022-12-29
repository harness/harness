// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

var _ store.ServiceAccountStore = (*ServiceAccountStore)(nil)

// serviceAccount is a DB representation of a service account principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type serviceAccount struct {
	types.ServiceAccount
	UIDUnique string `db:"principal_uidUnique"`
}

// NewServiceAccountStore returns a new ServiceAccountStore.
func NewServiceAccountStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) *ServiceAccountStore {
	return &ServiceAccountStore{
		db:                db,
		uidTransformation: uidTransformation,
	}
}

// ServiceAccountStore implements a ServiceAccountStore backed by a relational
// database.
type ServiceAccountStore struct {
	db                *sqlx.DB
	uidTransformation store.PrincipalUIDTransformation
}

// Find finds the service account by id.
func (s *ServiceAccountStore) Find(ctx context.Context, id int64) (*types.ServiceAccount, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(serviceAccount)
	if err := db.GetContext(ctx, dst, serviceAccountSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}
	return s.mapDBServiceAccount(dst), nil
}

// FindUID finds the service account by uid.
func (s *ServiceAccountStore) FindUID(ctx context.Context, uid string) (*types.ServiceAccount, error) {
	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(enum.PrincipalTypeServiceAccount, uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(serviceAccount)
	if err = db.GetContext(ctx, dst, serviceAccountSelectUIDUnique, uidUnique); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}

	return s.mapDBServiceAccount(dst), nil
}

// Create saves the service account.
func (s *ServiceAccountStore) Create(ctx context.Context, sa *types.ServiceAccount) error {
	dbSA, err := s.mapToDBserviceAccount(sa)
	if err != nil {
		return fmt.Errorf("failed to map db service account: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(serviceAccountInsert, dbSA)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service account object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&sa.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the service account details.
func (s *ServiceAccountStore) Update(ctx context.Context, sa *types.ServiceAccount) error {
	dbSA, err := s.mapToDBserviceAccount(sa)
	if err != nil {
		return fmt.Errorf("failed to map db service account: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(serviceAccountUpdate, dbSA)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind service account object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the service account.
func (s *ServiceAccountStore) Delete(ctx context.Context, id int64) error {
	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, serviceAccountDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	return nil
}

// List returns a list of service accounts for a specific parent.
func (s *ServiceAccountStore) List(ctx context.Context, parentType enum.ParentResourceType,
	parentID int64) ([]*types.ServiceAccount, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*serviceAccount{}
	err := db.SelectContext(ctx, &dst, serviceAccountSelectByParentTypeAndID, parentType, parentID)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed executing default list query")
	}

	return s.mapDBServiceAccounts(dst), nil
}

// Count returns a count of service accounts for a specific parent.
func (s *ServiceAccountStore) Count(ctx context.Context,
	parentType enum.ParentResourceType, parentID int64) (int64, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err := db.QueryRowContext(ctx, serviceAccountCountByParentTypeAndID, parentType, parentID).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

func (s *ServiceAccountStore) mapDBServiceAccount(dbSA *serviceAccount) *types.ServiceAccount {
	return &dbSA.ServiceAccount
}

func (s *ServiceAccountStore) mapDBServiceAccounts(dbSAs []*serviceAccount) []*types.ServiceAccount {
	res := make([]*types.ServiceAccount, len(dbSAs))
	for i := range dbSAs {
		res[i] = s.mapDBServiceAccount(dbSAs[i])
	}
	return res
}

func (s *ServiceAccountStore) mapToDBserviceAccount(sa *types.ServiceAccount) (*serviceAccount, error) {
	// service account comes from outside.
	if sa == nil {
		return nil, fmt.Errorf("service account is nil")
	}

	uidUnique, err := s.uidTransformation(enum.PrincipalTypeServiceAccount, sa.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform service account UID: %w", err)
	}
	dbSA := &serviceAccount{
		ServiceAccount: *sa,
		UIDUnique:      uidUnique,
	}

	return dbSA, nil
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
,principal_uidUnique
,principal_email
,principal_displayName
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
ORDER BY principal_uid ASC
`

const serviceAccountSelectID = serviceAccountBase + `
WHERE principal_type = "serviceaccount" AND principal_id = $1
`

const serviceAccountSelectUIDUnique = serviceAccountBase + `
WHERE principal_type = "serviceaccount" AND principal_uidUnique = $1
`

const serviceAccountDelete = `
DELETE FROM principals
WHERE principal_type = "serviceaccount" AND principal_id = $1
`

const serviceAccountInsert = `
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
,principal_sa_parentType
,principal_sa_parentId
) values (
 "serviceaccount"
,:principal_uid
,:principal_uidUnique
,:principal_email
,:principal_displayName
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
principal_email     	  = :principal_email
,principal_displayName    = :principal_displayName
,principal_blocked        = :principal_blocked
,principal_salt           = :principal_salt
,principal_updated        = :principal_updated
WHERE principal_type = "serviceaccount" AND principal_id = :principal_id
`
