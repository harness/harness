// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"strings"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
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
		return nil, processSQLErrorf(err, "Select by id query failed")
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
		return nil, store.ErrResourceNotFound
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(principal)
	if err = db.GetContext(ctx, dst, sqlQuery, uidUnique); err != nil {
		return nil, processSQLErrorf(err, "Select by uid query failed")
	}

	return s.mapDBPrincipal(dst), nil
}

// FindByEmail finds the principal by email.
func (s *PrincipalStore) FindByEmail(ctx context.Context, email string) (*types.Principal, error) {
	const sqlQuery = principalSelectBase + `
		WHERE LOWER(principal_email) = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(principal)
	if err := db.GetContext(ctx, dst, sqlQuery, strings.ToLower(email)); err != nil {
		return nil, processSQLErrorf(err, "Select by email query failed")
	}

	return s.mapDBPrincipal(dst), nil
}

func (s *PrincipalStore) mapDBPrincipal(dbPrincipal *principal) *types.Principal {
	return &dbPrincipal.Principal
}
