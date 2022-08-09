// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"strconv"

	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/types"
	"github.com/bradrydzewski/my-app/types/enum"

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
	err := s.db.Get(dst, userSelectID, id)
	return dst, err
}

// FindEmail finds the user by email.
func (s *UserStore) FindEmail(ctx context.Context, email string) (*types.User, error) {
	dst := new(types.User)
	err := s.db.Get(dst, userSelectEmail, email)
	return dst, err
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
func (s *UserStore) List(ctx context.Context, opts types.UserFilter) ([]*types.User, error) {
	dst := []*types.User{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.UserAttrNone {
		err := s.db.Select(&dst, userSelect, limit(opts.Size), offset(opts.Page, opts.Size))
		return dst, err
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
		return dst, err
	}

	err = s.db.Select(&dst, sql)
	return dst, err
}

// Create saves the user details.
func (s *UserStore) Create(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userInsert, user)
	if err != nil {
		return err
	}
	return s.db.QueryRow(query, arg...).Scan(&user.ID)
}

// Update updates the user details.
func (s *UserStore) Update(ctx context.Context, user *types.User) error {
	query, arg, err := s.db.BindNamed(userUpdate, user)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(query, arg...)
	return err
}

// Delete deletes the user.
func (s *UserStore) Delete(ctx context.Context, user *types.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// delete the user
	if _, err := tx.Exec(userDelete, user.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// Count returns a count of users.
func (s *UserStore) Count(context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRow(userCount).Scan(&count)
	return count, err
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
