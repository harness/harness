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
	"strings"

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// user is a DB representation of a user principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type user struct {
	types.User
	UIDUnique string `db:"principal_uid_unique"`
}

const userColumns = principalCommonColumns + `
	,principal_user_password`

const userSelectBase = `
	SELECT` + userColumns + `
	FROM principals`

// FindUser finds the user by id.
func (s *PrincipalStore) FindUser(ctx context.Context, id int64) (*types.User, error) {
	const sqlQuery = userSelectBase + `
		WHERE principal_type = 'user' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(user)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by id query failed")
	}

	return s.mapDBUser(dst), nil
}

// FindUserByUID finds the user by uid.
func (s *PrincipalStore) FindUserByUID(ctx context.Context, uid string) (*types.User, error) {
	const sqlQuery = userSelectBase + `
		WHERE principal_type = 'user' AND principal_uid_unique = $1`

	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, gitness_store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(user)
	if err = db.GetContext(ctx, dst, sqlQuery, uidUnique); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by uid query failed")
	}

	return s.mapDBUser(dst), nil
}

// FindUserByEmail finds the user by email.
func (s *PrincipalStore) FindUserByEmail(ctx context.Context, email string) (*types.User, error) {
	const sqlQuery = userSelectBase + `
		WHERE principal_type = 'user' AND LOWER(principal_email) = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(user)
	if err := db.GetContext(ctx, dst, sqlQuery, strings.ToLower(email)); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by email query failed")
	}

	return s.mapDBUser(dst), nil
}

// CreateUser saves the user details.
func (s *PrincipalStore) CreateUser(ctx context.Context, user *types.User) error {
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
			,principal_user_password
		) values (
			'user'
			,:principal_uid
			,:principal_uid_unique
			,:principal_email
			,:principal_display_name
			,:principal_admin
			,:principal_blocked
			,:principal_salt
			,:principal_created
			,:principal_updated
			,:principal_user_password
		) RETURNING principal_id`

	dbUser, err := s.mapToDBUser(user)
	if err != nil {
		return fmt.Errorf("failed to map db user: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbUser)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind user object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&user.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// UpdateUser updates an existing user.
func (s *PrincipalStore) UpdateUser(ctx context.Context, user *types.User) error {
	const sqlQuery = `
		UPDATE principals
		SET
			 principal_uid	          = :principal_uid
			,principal_uid_unique     = :principal_uid_unique
			,principal_email          = :principal_email
			,principal_display_name   = :principal_display_name
			,principal_admin          = :principal_admin
			,principal_blocked        = :principal_blocked
			,principal_salt           = :principal_salt
			,principal_updated        = :principal_updated
			,principal_user_password  = :principal_user_password
		WHERE principal_type = 'user' AND principal_id = :principal_id`

	dbUser, err := s.mapToDBUser(user)
	if err != nil {
		return fmt.Errorf("failed to map db user: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbUser)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind user object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Update query failed")
	}

	return err
}

// DeleteUser deletes the user.
func (s *PrincipalStore) DeleteUser(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM principals
		WHERE principal_type = 'user' AND principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// ListUsers returns a list of users.
func (s *PrincipalStore) ListUsers(ctx context.Context, opts *types.UserFilter) ([]*types.User, error) {
	db := dbtx.GetAccessor(ctx, s.db)
	dst := []*user{}

	stmt := database.Builder.
		Select(userColumns).
		From("principals").
		Where("principal_type = 'user'")
	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	order := opts.Order
	if order == enum.OrderDefault {
		order = enum.OrderAsc
	}

	switch opts.Sort {
	case enum.UserAttrName, enum.UserAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("principal_display_name " + order.String())
	case enum.UserAttrCreated:
		stmt = stmt.OrderBy("principal_created " + order.String())
	case enum.UserAttrUpdated:
		stmt = stmt.OrderBy("principal_updated " + order.String())
	case enum.UserAttrEmail:
		stmt = stmt.OrderBy("LOWER(principal_email) " + order.String())
	case enum.UserAttrUID:
		stmt = stmt.OrderBy("principal_uid " + order.String())
	case enum.UserAttrAdmin:
		stmt = stmt.OrderBy("principal_admin " + order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapDBUsers(dst), nil
}

// CountUsers returns a count of users matching the given filter.
func (s *PrincipalStore) CountUsers(ctx context.Context, opts *types.UserFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("principals").
		Where("principal_type = 'user'")

	if opts.Admin {
		stmt = stmt.Where("principal_admin = ?", opts.Admin)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (s *PrincipalStore) mapDBUser(dbUser *user) *types.User {
	return &dbUser.User
}

func (s *PrincipalStore) mapDBUsers(dbUsers []*user) []*types.User {
	res := make([]*types.User, len(dbUsers))
	for i := range dbUsers {
		res[i] = s.mapDBUser(dbUsers[i])
	}
	return res
}

func (s *PrincipalStore) mapToDBUser(usr *types.User) (*user, error) {
	// user comes from outside.
	if usr == nil {
		return nil, fmt.Errorf("user is nil")
	}

	uidUnique, err := s.uidTransformation(usr.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform user UID: %w", err)
	}
	dbUser := &user{
		User:      *usr,
		UIDUnique: uidUnique,
	}

	return dbUser, nil
}
