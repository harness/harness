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

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// serviceAccount is a DB representation of a service account principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type serviceAccount struct {
	types.ServiceAccount
	UIDUnique string `db:"principal_uid_unique"`
}

const serviceAccountColumns = principalCommonColumns + `
	,principal_sa_parent_type
	,principal_sa_parent_id`

const serviceAccountSelectBase = `
	SELECT` + serviceAccountColumns + `
	FROM principals`

// FindServiceAccount finds the service account by id.
func (s *PrincipalStore) FindServiceAccount(ctx context.Context, id int64) (*types.ServiceAccount, error) {
	const sqlQuery = serviceAccountSelectBase + `
		WHERE principal_type = 'serviceaccount' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(serviceAccount)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by id query failed")
	}
	return s.mapDBServiceAccount(dst), nil
}

// FindServiceAccountByUID finds the service account by uid.
func (s *PrincipalStore) FindServiceAccountByUID(ctx context.Context, uid string) (*types.ServiceAccount, error) {
	const sqlQuery = serviceAccountSelectBase + `
		WHERE principal_type = 'serviceaccount' AND principal_uid_unique = $1`

	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, gitness_store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(serviceAccount)
	if err = db.GetContext(ctx, dst, sqlQuery, uidUnique); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by uid query failed")
	}

	return s.mapDBServiceAccount(dst), nil
}

// CreateServiceAccount saves the service account.
func (s *PrincipalStore) CreateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error {
	const sqlQuery = `
		INSERT INTO principals (
			principal_type
			,principal_uid
			,principal_uid_unique
			,principal_email
			,principal_display_name
			,principal_admin
			,principal_blocked
			,principal_salt
			,principal_created
			,principal_updated
			,principal_sa_parent_type
			,principal_sa_parent_id
		) values (
			'serviceaccount'
			,:principal_uid
			,:principal_uid_unique
			,:principal_email
			,:principal_display_name
			,false
			,:principal_blocked
			,:principal_salt
			,:principal_created
			,:principal_updated
			,:principal_sa_parent_type
			,:principal_sa_parent_id
		) RETURNING principal_id`

	dbSA, err := s.mapToDBserviceAccount(sa)
	if err != nil {
		return fmt.Errorf("failed to map db service account: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSA)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind service account object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&sa.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// UpdateServiceAccount updates the service account details.
func (s *PrincipalStore) UpdateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error {
	const sqlQuery = `
		UPDATE principals
		SET
			 principal_uid	          = :principal_uid
			,principal_uid_unique     = :principal_uid_unique
			,principal_email          = :principal_email
			,principal_display_name   = :principal_display_name
			,principal_blocked        = :principal_blocked
			,principal_salt           = :principal_salt
			,principal_updated        = :principal_updated
		WHERE principal_type = 'serviceaccount' AND principal_id = :principal_id`

	dbSA, err := s.mapToDBserviceAccount(sa)
	if err != nil {
		return fmt.Errorf("failed to map db service account: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSA)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind service account object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Update query failed")
	}

	return err
}

// DeleteServiceAccount deletes the service account.
func (s *PrincipalStore) DeleteServiceAccount(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM principals
		WHERE principal_type = 'serviceaccount' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// ListServiceAccounts returns a list of service accounts for a specific parent.
func (s *PrincipalStore) ListServiceAccounts(ctx context.Context, parentType enum.ParentResourceType,
	parentID int64) ([]*types.ServiceAccount, error) {
	const sqlQuery = serviceAccountSelectBase + `
		WHERE principal_type = 'serviceaccount' AND principal_sa_parent_type = $1 AND principal_sa_parent_id = $2
		ORDER BY principal_uid ASC`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*serviceAccount{}
	err := db.SelectContext(ctx, &dst, sqlQuery, parentType, parentID)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing default list query")
	}

	return s.mapDBServiceAccounts(dst), nil
}

// CountServiceAccounts returns a count of service accounts for a specific parent.
func (s *PrincipalStore) CountServiceAccounts(ctx context.Context,
	parentType enum.ParentResourceType, parentID int64) (int64, error) {
	const sqlQuery = `
		SELECT count(*)
		FROM principals
		WHERE principal_type = 'serviceaccount' and principal_sa_parentType = $1 and principal_sa_parentId = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err := db.QueryRowContext(ctx, sqlQuery, parentType, parentID).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (s *PrincipalStore) mapDBServiceAccount(dbSA *serviceAccount) *types.ServiceAccount {
	return &dbSA.ServiceAccount
}

func (s *PrincipalStore) mapDBServiceAccounts(dbSAs []*serviceAccount) []*types.ServiceAccount {
	res := make([]*types.ServiceAccount, len(dbSAs))
	for i := range dbSAs {
		res[i] = s.mapDBServiceAccount(dbSAs[i])
	}
	return res
}

func (s *PrincipalStore) mapToDBserviceAccount(sa *types.ServiceAccount) (*serviceAccount, error) {
	// service account comes from outside.
	if sa == nil {
		return nil, fmt.Errorf("service account is nil")
	}

	uidUnique, err := s.uidTransformation(sa.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform service account UID: %w", err)
	}
	dbSA := &serviceAccount{
		ServiceAccount: *sa,
		UIDUnique:      uidUnique,
	}

	return dbSA, nil
}
