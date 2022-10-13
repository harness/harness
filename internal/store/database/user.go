// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"database/sql"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"

	"github.com/jmoiron/sqlx"
)

var _ store.UserStore = (*UserStore)(nil)

// NewUserStore returns a new UserStore.
func NewUserStore(db *sqlx.DB) *UserStore {
	return &UserStore{db}
}

// UserStore implements a UserStore backed by a relational
// database.
type UserStore struct {
	db *sqlx.DB
}

// Find finds the user by id.
func (s *UserStore) Find(ctx context.Context, id int64) (*types.User, error) {
	dst := new(types.User)
	if err := s.db.GetContext(ctx, dst, userSelectID, id); err != nil {
		return nil, processSQLErrorf(err, "Select by id query failed")
	}
	return dst, nil
}

// FindUID finds the user by uid.
func (s *UserStore) FindUID(ctx context.Context, uid string) (*types.User, error) {
	dst := new(types.User)
	if err := s.db.GetContext(ctx, dst, userSelectUID, uid); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}
	return dst, nil
}

// FindEmail finds the user by email.
func (s *UserStore) FindEmail(ctx context.Context, email string) (*types.User, error) {
	dst := new(types.User)
	if err := s.db.GetContext(ctx, dst, userSelectEmail, email); err != nil {
		return nil, processSQLErrorf(err, "Select by email query failed")
	}
	return dst, nil
}

// List returns a list of users.
func (s *UserStore) List(ctx context.Context, opts *types.UserFilter) ([]*types.User, error) {
	dst := []*types.User{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.UserAttrNone {
		err := s.db.SelectContext(ctx, &dst, userSelect, limit(opts.Size), offset(opts.Page, opts.Size))
		if err != nil {
			return nil, processSQLErrorf(err, "Failed executing default list query")
		}
		return dst, nil
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
		stmt = stmt.OrderBy("principal_name " + opts.Order.String())
	case enum.UserAttrCreated:
		stmt = stmt.OrderBy("principal_created " + opts.Order.String())
	case enum.UserAttrUpdated:
		stmt = stmt.OrderBy("principal_updated " + opts.Order.String())
	case enum.UserAttrEmail:
		stmt = stmt.OrderBy("principal_user_email " + opts.Order.String())
	case enum.UserAttrUID:
		stmt = stmt.OrderBy("principal_uid " + opts.Order.String())
	case enum.UserAttrAdmin:
		stmt = stmt.OrderBy("principal_admin " + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = s.db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Create saves the user details.
func (s *UserStore) Create(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userInsert, user)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind user object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&user.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the user details.
func (s *UserStore) Update(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userUpdate, user)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind user object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the user.
func (s *UserStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)
	// delete the user
	if _, err = tx.ExecContext(ctx, userDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}
	return tx.Commit()
}

// Count returns a count of users.
func (s *UserStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, userCount).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

const userCount = `
SELECT count(*)
FROM principals
WHERE principal_type = "user"
`

const userBase = `
SELECT
principal_id
,principal_uid
,principal_name
,principal_admin
,principal_blocked
,principal_salt
,principal_created
,principal_updated
,principal_user_email
,principal_user_password
FROM principals
`

const userSelect = userBase + `
WHERE principal_type = "user"
ORDER BY principal_name ASC
LIMIT $1 OFFSET $2
`

const userSelectID = userBase + `
WHERE principal_type = "user" AND principal_id = $1
`

const userSelectUID = userBase + `
WHERE principal_type = "user" AND principal_uid = $1
`

const userSelectEmail = userBase + `
WHERE principal_type = "user" AND principal_user_email = $1
`

const userDelete = `
DELETE FROM principals
WHERE principal_type = "user" AND principal_id = $1
`

const userInsert = `
INSERT INTO principals (
principal_type
,principal_uid
,principal_name
,principal_admin
,principal_blocked
,principal_salt
,principal_created
,principal_updated
,principal_user_email
,principal_user_password
) values (
"user"
,:principal_uid
,:principal_name
,:principal_admin
,:principal_blocked
,:principal_salt
,:principal_created
,:principal_updated
,:principal_user_email
,:principal_user_password
) RETURNING principal_id
`

const userUpdate = `
UPDATE principals
SET
 principal_name    = :principal_name
,principal_admin          = :principal_admin
,principal_blocked        = :principal_blocked
,principal_salt           = :principal_salt
,principal_updated        = :principal_updated
,principal_user_email     = :principal_user_email
,principal_user_password  = :principal_user_password
WHERE principal_type = "user" AND principal_id = :principal_id
`
