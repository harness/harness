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
	"strings"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ store.PrincipalStore = (*PrincipalStore)(nil)

// NewPrincipalStore returns a new PrincipalStore.
func NewPrincipalStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) *PrincipalStore {
	return &PrincipalStore{
		db:                db,
		uidTransformation: uidTransformation,
	}
}

// PrincipalStore implements a PrincipalStore backed by a relational database.
type PrincipalStore struct {
	db                *sqlx.DB
	uidTransformation store.PrincipalUIDTransformation
}

// principal is a DB representation of a principal.
// It is required to allow storing transformed UIDs used for uniquness constraints and searching.
type principal struct {
	types.Principal
	UIDUnique string `db:"principal_uid_unique"`
}

// principalCommonColumns defines the columns that are the same across all principals.
const principalCommonColumns = `
	principal_id
	,principal_uid
	,principal_uid_unique
	,principal_email
	,principal_display_name
	,principal_admin
	,principal_blocked
	,principal_salt
	,principal_created
	,principal_updated`

// principalColumns defines the column that are used only in a principal itself
// (for explicit principals the type is implicit, only the generic principal struct stores it explicitly).
const principalColumns = principalCommonColumns + `
	,principal_type`

//nolint:goconst
const principalSelectBase = `
	SELECT` + principalColumns + `
	FROM principals`

// Find finds the principal by id.
func (s *PrincipalStore) Find(ctx context.Context, id int64) (*types.Principal, error) {
	const sqlQuery = principalSelectBase + `
		WHERE principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(principal)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by id query failed")
	}

	return s.mapDBPrincipal(dst), nil
}

// FindByUID finds the principal by uid.
func (s *PrincipalStore) FindByUID(ctx context.Context, uid string) (*types.Principal, error) {
	const sqlQuery = principalSelectBase + `
		WHERE principal_uid_unique = $1`

	// map the UID to unique UID before searching!
	uidUnique, err := s.uidTransformation(uid)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform uid '%s': %s", uid, err.Error())
		return nil, gitness_store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(principal)
	if err = db.GetContext(ctx, dst, sqlQuery, uidUnique); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by uid query failed")
	}

	return s.mapDBPrincipal(dst), nil
}

// FindManyByUID returns all principals found for the provided UIDs.
// If a UID isn't found, it's not returned in the list.
func (s *PrincipalStore) FindManyByUID(ctx context.Context, uids []string) ([]*types.Principal, error) {
	// map the UIDs to unique UIDs before searching!
	uniqueUIDs := make([]string, len(uids))
	for i := range uids {
		var err error
		uniqueUIDs[i], err = s.uidTransformation(uids[i])
		if err != nil {
			// in case we fail to transform, skip the entry (as it can't exist in the first place)
			log.Ctx(ctx).Warn().Msgf("failed to transform uid '%s': %s", uids[i], err.Error())
		}
	}

	stmt := database.Builder.
		Select(principalColumns).
		From("principals").
		Where(squirrel.Eq{"principal_uid_unique": uids})
	db := dbtx.GetAccessor(ctx, s.db)

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate find many principal query")
	}

	dst := []*principal{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "find many by uid for principals query failed")
	}

	return s.mapDBPrincipals(dst), nil
}

// FindByEmail finds the principal by email.
func (s *PrincipalStore) FindByEmail(ctx context.Context, email string) (*types.Principal, error) {
	const sqlQuery = principalSelectBase + `
		WHERE LOWER(principal_email) = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(principal)
	if err := db.GetContext(ctx, dst, sqlQuery, strings.ToLower(email)); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select by email query failed")
	}

	return s.mapDBPrincipal(dst), nil
}

func (s *PrincipalStore) FindManyByEmail(
	ctx context.Context,
	emails []string,
) ([]*types.Principal, error) {
	lowerCaseEmails := make([]string, len(emails))
	for i := range emails {
		lowerCaseEmails[i] = strings.ToLower(emails[i])
	}

	stmt := database.Builder.
		Select(principalColumns).
		From("principals").
		Where(squirrel.Eq{"principal_email": lowerCaseEmails})

	db := dbtx.GetAccessor(ctx, s.db)

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate find many principal query")
	}

	dst := []*principal{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "find many by email for principal query failed")
	}

	return s.mapDBPrincipals(dst), nil
}

// List lists the principals matching the provided filter.
func (s *PrincipalStore) List(ctx context.Context,
	opts *types.PrincipalFilter) ([]*types.Principal, error) {
	stmt := database.Builder.
		Select(principalColumns).
		From("principals")

	if len(opts.Types) == 1 {
		stmt = stmt.Where("principal_type = ?", opts.Types[0])
	} else if len(opts.Types) > 1 {
		stmt = stmt.Where(squirrel.Eq{"principal_type": opts.Types})
	}

	if opts.Query != "" {
		// TODO: optimize performance
		// https://harness.atlassian.net/browse/CODE-522
		stmt = stmt.Where(squirrel.Or{
			squirrel.Expr(PartialMatch("principal_uid", opts.Query)),
			squirrel.Expr(PartialMatch("principal_email", opts.Query)),
			squirrel.Expr(PartialMatch("principal_display_name", opts.Query)),
		})
	}

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*principal{}
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Search by display_name and email query failed")
	}

	return s.mapDBPrincipals(dst), nil
}

func (s *PrincipalStore) mapDBPrincipal(dbPrincipal *principal) *types.Principal {
	return &dbPrincipal.Principal
}

func (s *PrincipalStore) mapDBPrincipals(dbPrincipals []*principal) []*types.Principal {
	res := make([]*types.Principal, len(dbPrincipals))
	for i := range dbPrincipals {
		res[i] = s.mapDBPrincipal(dbPrincipals[i])
	}
	return res
}
