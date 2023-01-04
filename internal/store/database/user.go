// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ store.UserStore = (*UserStore)(nil)

// user is a DB representation of a user principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type user struct {
	types.User
	UIDUnique string `db:"principal_uid_unique"`
}

// NewUserStore returns a new UserStore.
func NewUserStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) *UserStore {
	return &UserStore{
		db:                db,
		uidTransformation: uidTransformation,
	}
}

// UserStore implements a UserStore backed by a relational database.
type UserStore struct {
	db                *sqlx.DB
	uidTransformation store.PrincipalUIDTransformation
}

// Find finds the user by id.
func (s *UserStore) Find(ctx context.Context, id int64) (*types.User, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(user)
	if err := db.GetContext(ctx, dst, userSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}

	return s.mapDBUser(dst), nil
}

// FindUID finds the user by uid.
func (s *UserStore) FindUID(ctx context.Context, uid string) (*types.User, error) {
	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(enum.PrincipalTypeUser, uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(user)
	if err = db.GetContext(ctx, dst, userSelectUIDUnique, uidUnique); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}

	return s.mapDBUser(dst), nil
}

// FindEmail finds the user by email.
func (s *UserStore) FindEmail(ctx context.Context, email string) (*types.User, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	const sqlQuery = userBase + `
	WHERE principal_type = 'user' AND LOWER(principal_email) = $1`

	dst := new(user)
	if err := db.GetContext(ctx, dst, sqlQuery, strings.ToLower(email)); err != nil {
		return nil, processSQLErrorf(err, "Select by email query failed")
	}

	return s.mapDBUser(dst), nil
}

// Create saves the user details.
func (s *UserStore) Create(ctx context.Context, user *types.User) error {
	dbUser, err := s.mapToDBUser(user)
	if err != nil {
		return fmt.Errorf("failed to map db user: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(userInsert, dbUser)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind user object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&user.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the user details.
func (s *UserStore) Update(ctx context.Context, user *types.User) error {
	dbUser, err := s.mapToDBUser(user)
	if err != nil {
		return fmt.Errorf("failed to map db user: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(userUpdate, dbUser)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind user object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the user.
func (s *UserStore) Delete(ctx context.Context, id int64) error {
	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, userDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	return nil
}

// List returns a list of users.
func (s *UserStore) List(ctx context.Context, opts *types.UserFilter) ([]*types.User, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*user{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.UserAttrNone {
		err := db.SelectContext(ctx, &dst, userSelect, limit(opts.Size), offset(opts.Page, opts.Size))
		if err != nil {
			return nil, processSQLErrorf(err, "Failed executing default list query")
		}
		return s.mapDBUsers(dst), nil
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("users")
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.UserAttrName, enum.UserAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("principal_display_name " + opts.Order.String())
	case enum.UserAttrCreated:
		stmt = stmt.OrderBy("principal_created " + opts.Order.String())
	case enum.UserAttrUpdated:
		stmt = stmt.OrderBy("principal_updated " + opts.Order.String())
	case enum.UserAttrEmail:
		stmt = stmt.OrderBy("LOWER(principal_email) " + opts.Order.String())
	case enum.UserAttrUID:
		stmt = stmt.OrderBy("principal_uid " + opts.Order.String())
	case enum.UserAttrAdmin:
		stmt = stmt.OrderBy("principal_admin " + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return s.mapDBUsers(dst), nil
}

// Count returns a count of users.
func (s *UserStore) Count(ctx context.Context) (int64, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err := db.QueryRowContext(ctx, userCount).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

func (s *UserStore) mapDBUser(dbUser *user) *types.User {
	return &dbUser.User
}

func (s *UserStore) mapDBUsers(dbUsers []*user) []*types.User {
	res := make([]*types.User, len(dbUsers))
	for i := range dbUsers {
		res[i] = s.mapDBUser(dbUsers[i])
	}
	return res
}

func (s *UserStore) mapToDBUser(usr *types.User) (*user, error) {
	// user comes from outside.
	if usr == nil {
		return nil, fmt.Errorf("user is nil")
	}

	uidUnique, err := s.uidTransformation(enum.PrincipalTypeUser, usr.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to transform user UID: %w", err)
	}
	dbUser := &user{
		User:      *usr,
		UIDUnique: uidUnique,
	}

	return dbUser, nil
}

const userCount = `
SELECT count(*)
FROM principals
WHERE principal_type = 'user'
`

const userBase = `
SELECT
principal_id
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
FROM principals
`

const userSelect = userBase + `
WHERE principal_type = 'user'
ORDER BY principal_uid ASC
LIMIT $1 OFFSET $2
`

const userSelectID = userBase + `
WHERE principal_type = 'user' AND principal_id = $1
`

const userSelectUIDUnique = userBase + `
WHERE principal_type = 'user' AND principal_uid_unique = $1
`

const userDelete = `
DELETE FROM principals
WHERE principal_type = 'user' AND principal_id = $1
`

const userInsert = `
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
) RETURNING principal_id
`

const userUpdate = `
UPDATE principals
SET
principal_email     	  = :principal_email
,principal_display_name   = :principal_display_name
,principal_admin          = :principal_admin
,principal_blocked        = :principal_blocked
,principal_salt           = :principal_salt
,principal_updated        = :principal_updated
,principal_user_password  = :principal_user_password
WHERE principal_type = 'user' AND principal_id = :principal_id
`
