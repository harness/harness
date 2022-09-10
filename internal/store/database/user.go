// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"strconv"

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
		return nil, processSqlErrorf(err, "Select by id query failed")
	}
	return dst, nil
}

// FindEmail finds the user by email.
func (s *UserStore) FindEmail(ctx context.Context, email string) (*types.User, error) {
	dst := new(types.User)
	if err := s.db.GetContext(ctx, dst, userSelectEmail, email); err != nil {
		return nil, processSqlErrorf(err, "Select by email query failed")
	}
	return dst, nil
}

// FindKey finds the user unique key (email or id).
func (s *UserStore) FindKey(ctx context.Context, key string) (*types.User, error) {
	id, err := strconv.ParseInt(key, 10, 64)
	if err == nil {
		return s.Find(ctx, id)
	} else {
		return s.FindEmail(ctx, key)
	}
}

// List returns a list of users.
func (s *UserStore) List(ctx context.Context, opts *types.UserFilter) ([]*types.User, error) {
	dst := []*types.User{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.UserAttrNone {
		err := s.db.SelectContext(ctx, &dst, userSelect, limit(opts.Size), offset(opts.Page, opts.Size))
		if err != nil {
			return nil, processSqlErrorf(err, "Failed executing default list query")
		}
		return dst, nil
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("users")
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.UserAttrCreated:
		// NOTE: string concatination is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("user_id " + opts.Order.String())
	case enum.UserAttrUpdated:
		stmt = stmt.OrderBy("user_updated " + opts.Order.String())
	case enum.UserAttrEmail:
		stmt = stmt.OrderBy("user_email " + opts.Order.String())
	case enum.UserAttrId:
		stmt = stmt.OrderBy("user_id " + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = s.db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, processSqlErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Create saves the user details.
func (s *UserStore) Create(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userInsert, user)
	if err != nil {
		return processSqlErrorf(err, "Failed to bind user object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&user.ID); err != nil {
		return processSqlErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the user details.
func (s *UserStore) Update(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userUpdate, user)
	if err != nil {
		return processSqlErrorf(err, "Failed to bind user object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSqlErrorf(err, "Update query failed")
	}

	return err
}

// Delete deletes the user.
func (s *UserStore) Delete(ctx context.Context, user *types.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return processSqlErrorf(err, "Failed to start a new transaction")
	}
	defer tx.Rollback()
	// delete the user
	if _, err := tx.ExecContext(ctx, userDelete, user.ID); err != nil {
		return processSqlErrorf(err, "The delete query failed")
	}
	return tx.Commit()
}

// Count returns a count of users.
func (s *UserStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, userCount).Scan(&count)
	if err != nil {
		return 0, processSqlErrorf(err, "Failed executing count query")
	}
	return count, nil
}

const userCount = `
SELECT count(*)
FROM users
`

const userBase = `
SELECT
 user_id
,user_email
,user_name
,user_company
,user_password
,user_salt
,user_admin
,user_blocked
,user_created
,user_updated
,user_authed
FROM users
`

const userSelect = userBase + `
ORDER BY user_email ASC
LIMIT $1 OFFSET $2
`

const userSelectID = userBase + `
WHERE user_id = $1
`

const userSelectEmail = userBase + `
WHERE user_email = $1
`

const userSelectToken = userBase + `
WHERE user_salt = $1
`

const userDelete = `
DELETE FROM users
WHERE user_id = $1
`

const userInsert = `
INSERT INTO users (
 user_email
,user_name
,user_company
,user_password
,user_salt
,user_admin
,user_blocked
,user_created
,user_updated
,user_authed
) values (
 :user_email
,:user_name
,:user_company
,:user_password
,:user_salt
,:user_admin
,:user_blocked
,:user_created
,:user_updated
,:user_authed
) RETURNING user_id
`

const userUpdate = `
UPDATE users
SET
 user_email     = :user_email
,user_name      = :user_name
,user_company   = :user_company
,user_password  = :user_password
,user_salt     = :user_salt
,user_admin     = :user_admin
,user_blocked   = :user_blocked
,user_created   = :user_created
,user_updated   = :user_updated
,user_authed    = :user_authed
WHERE user_id = :user_id
`
