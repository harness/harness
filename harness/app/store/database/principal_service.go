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

	"github.com/rs/zerolog/log"
)

// service is a DB representation of a service principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type service struct {
	types.Service
	UIDUnique string `db:"principal_uid_unique"`
}

// service doesn't have any extra columns.
const serviceColumns = principalCommonColumns

const serviceSelectBase = `
	SELECT` + serviceColumns + `
	FROM principals`

// FindService finds the service by id.
func (s *PrincipalStore) FindService(ctx context.Context, id int64) (*types.Service, error) {
	const sqlQuery = serviceSelectBase + `
		WHERE principal_type = 'service' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(service)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by id query failed")
	}

	return s.mapDBService(dst), nil
}

// FindServiceByUID finds the service by uid.
func (s *PrincipalStore) FindServiceByUID(ctx context.Context, uid string) (*types.Service, error) {
	const sqlQuery = serviceSelectBase + `
		WHERE principal_type = 'service' AND principal_uid_unique = $1`

	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, gitness_store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(service)
	if err = db.GetContext(ctx, dst, sqlQuery, uidUnique); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by uid query failed")
	}

	return s.mapDBService(dst), nil
}

// CreateService saves the service.
func (s *PrincipalStore) CreateService(ctx context.Context, svc *types.Service) error {
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
		) values (
			'service'
			,:principal_uid
			,:principal_uid_unique
			,:principal_email
			,:principal_display_name
			,:principal_admin
			,:principal_blocked
			,:principal_salt
			,:principal_created
			,:principal_updated
		) RETURNING principal_id`

	dbSVC, err := s.mapToDBservice(svc)
	if err != nil {
		return fmt.Errorf("failed to map db service: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSVC)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind service object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&svc.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// UpdateService updates the service.
func (s *PrincipalStore) UpdateService(ctx context.Context, svc *types.Service) error {
	const sqlQuery = `
		UPDATE principals
		SET
			 principal_uid	          = :principal_uid
			,principal_uid_unique     = :principal_uid_unique
			,principal_email          = :principal_email
			,principal_display_name   = :principal_display_name
			,principal_admin          = :principal_admin
			,principal_blocked        = :principal_blocked
			,principal_updated        = :principal_updated
		WHERE principal_type = 'service' AND principal_id = :principal_id`

	dbSVC, err := s.mapToDBservice(svc)
	if err != nil {
		return fmt.Errorf("failed to map db service: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSVC)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind service object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Update query failed")
	}

	return err
}

// DeleteService deletes the service.
func (s *PrincipalStore) DeleteService(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM principals
		WHERE principal_type = 'service' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	// delete the service
	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// ListServices returns a list of service for a specific parent.
func (s *PrincipalStore) ListServices(ctx context.Context) ([]*types.Service, error) {
	const sqlQuery = serviceSelectBase + `
		WHERE principal_type = 'service'
		ORDER BY principal_uid ASC`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*service{}

	err := db.SelectContext(ctx, &dst, sqlQuery)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing default list query")
	}

	return s.mapDBServices(dst), nil
}

// CountServices returns a count of service for a specific parent.
func (s *PrincipalStore) CountServices(ctx context.Context) (int64, error) {
	const sqlQuery = `
		SELECT count(*)
		FROM principals
		WHERE principal_type = 'service'`

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err := db.QueryRowContext(ctx, sqlQuery).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (s *PrincipalStore) mapDBService(dbSvc *service) *types.Service {
	return &dbSvc.Service
}

func (s *PrincipalStore) mapDBServices(dbSVCs []*service) []*types.Service {
	res := make([]*types.Service, len(dbSVCs))
	for i := range dbSVCs {
		res[i] = s.mapDBService(dbSVCs[i])
	}
	return res
}

func (s *PrincipalStore) mapToDBservice(svc *types.Service) (*service, error) {
	// service comes from outside.
	if svc == nil {
		return nil, fmt.Errorf("service is nil")
	}

	uidUnique, err := s.uidTransformation(svc.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform service UID: %w", err)
	}
	dbService := &service{
		Service:   *svc,
		UIDUnique: uidUnique,
	}

	return dbService, nil
}
