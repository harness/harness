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

	"github.com/Masterminds/squirrel"
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

func (s *PrincipalStore) FindManyServiceAccountByUID(
	ctx context.Context,
	uids []string,
) ([]*types.ServiceAccount, error) {
	uniqueUIDs := make([]string, len(uids))
	var err error
	for i, uid := range uids {
		uniqueUIDs[i], err = s.uidTransformation(uid)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
			return nil, gitness_store.ErrResourceNotFound
		}
	}

	stmt := database.Builder.
		Select(serviceAccountColumns).
		From("principals").
		Where("principal_type = ?", enum.PrincipalTypeServiceAccount).
		Where(squirrel.Eq{"principal_uid_unique": uniqueUIDs})
	db := dbtx.GetAccessor(ctx, s.db)

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate find many service accounts query")
	}

	dst := []*serviceAccount{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "find many service accounts failed")
	}

	return s.mapDBServiceAccounts(dst), nil
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
func (s *PrincipalStore) ListServiceAccounts(
	ctx context.Context,
	parentInfos []*types.ServiceAccountParentInfo,
	opts *types.PrincipalFilter,
) ([]*types.ServiceAccount, error) {
	stmt := database.Builder.
		Select(serviceAccountColumns).
		From("principals").
		Where("principal_type = ?", enum.PrincipalTypeServiceAccount)

	stmt, err := selectServiceAccountParents(parentInfos, stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to select service account parents: %w", err)
	}

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	if opts.Query != "" {
		stmt = stmt.Where(PartialMatch("principal_display_name", opts.Query))
	}

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(
			ctx, err, "failed to generate list service accounts query",
		)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*serviceAccount{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing default list query")
	}

	return s.mapDBServiceAccounts(dst), nil
}

// CountServiceAccounts returns a count of service accounts for a specific parent.
func (s *PrincipalStore) CountServiceAccounts(
	ctx context.Context,
	parentInfos []*types.ServiceAccountParentInfo,
	opts *types.PrincipalFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("principals").
		Where("principal_type = ?", enum.PrincipalTypeServiceAccount)

	if opts.Query != "" {
		stmt = stmt.Where(PartialMatch("principal_display_name", opts.Query))
	}

	stmt, err := selectServiceAccountParents(parentInfos, stmt)
	if err != nil {
		return 0, fmt.Errorf("failed to select service account parents: %w", err)
	}

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return 0, database.ProcessSQLErrorf(
			ctx, err, "failed to generate count service accounts query",
		)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.QueryRowContext(ctx, sqlQuery, params...).Scan(&count); err != nil {
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

func selectServiceAccountParents(
	parents []*types.ServiceAccountParentInfo,
	stmt squirrel.SelectBuilder,
) (squirrel.SelectBuilder, error) {
	var typeSelector squirrel.Or
	for _, parent := range parents {
		switch parent.Type {
		case enum.ParentResourceTypeRepo:
			typeSelector = append(typeSelector, squirrel.Eq{
				"principal_sa_parent_type": enum.ParentResourceTypeRepo,
				"principal_sa_parent_id":   parent.ID,
			})
		case enum.ParentResourceTypeSpace:
			typeSelector = append(typeSelector, squirrel.Eq{
				"principal_sa_parent_type": enum.ParentResourceTypeSpace,
				"principal_sa_parent_id":   parent.ID,
			})
		default:
			return squirrel.SelectBuilder{}, fmt.Errorf("service account parent type '%s' is not supported", parent.Type)
		}
	}

	return stmt.Where(typeSelector), nil
}
