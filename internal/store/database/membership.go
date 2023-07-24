// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.MembershipStore = (*MembershipStore)(nil)

// NewMembershipStore returns a new MembershipStore.
func NewMembershipStore(db *sqlx.DB, pCache store.PrincipalInfoCache) *MembershipStore {
	return &MembershipStore{
		db:     db,
		pCache: pCache,
	}
}

// MembershipStore implements store.MembershipStore backed by a relational database.
type MembershipStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

type membership struct {
	SpaceID     int64 `db:"membership_space_id"`
	PrincipalID int64 `db:"membership_principal_id"`

	CreatedBy int64 `db:"membership_created_by"`
	Created   int64 `db:"membership_created"`
	Updated   int64 `db:"membership_updated"`

	Role enum.MembershipRole `db:"membership_role"`
}

const (
	membershipColumns = `
		 membership_space_id
		,membership_principal_id
		,membership_created_by
		,membership_created
		,membership_updated
		,membership_role`

	membershipSelectBase = `
	SELECT` + membershipColumns + `
	FROM memberships`
)

// Find finds the membership by space id and principal id.
func (s *MembershipStore) Find(ctx context.Context, key types.MembershipKey) (*types.Membership, error) {
	const sqlQuery = membershipSelectBase + `
	WHERE membership_space_id = $1 AND membership_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &membership{}
	if err := db.GetContext(ctx, dst, sqlQuery, key.SpaceID, key.PrincipalID); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find membership")
	}

	return s.mapToMembership(ctx, dst), nil
}

// Create creates a new membership.
func (s *MembershipStore) Create(ctx context.Context, membership *types.Membership) error {
	const sqlQuery = `
	INSERT INTO memberships (
		 membership_space_id
		,membership_principal_id
		,membership_created_by
		,membership_created
		,membership_updated
		,membership_role
	) values (
		 :membership_space_id
		,:membership_principal_id
		,:membership_created_by
		,:membership_created
		,:membership_updated
		,:membership_role
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapToInternalMembership(membership))
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind membership object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(err, "Failed to insert membership")
	}

	return nil
}

// Update updates the role of a member of a space.
func (s *MembershipStore) Update(ctx context.Context, membership *types.Membership) error {
	const sqlQuery = `
	UPDATE memberships
	SET
		 membership_updated = :membership_updated
		,membership_role = :membership_role
	WHERE membership_space_id = :membership_space_id AND
	      membership_principal_id = :membership_principal_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbMembership := mapToInternalMembership(membership)
	dbMembership.Updated = time.Now().UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbMembership)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind membership object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update membership role")
	}

	membership.Updated = dbMembership.Updated

	return nil
}

// Delete deletes the membership.
func (s *MembershipStore) Delete(ctx context.Context, key types.MembershipKey) error {
	const sqlQuery = `
	DELETE from memberships
	WHERE membership_space_id = $1 AND
	      membership_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, key.SpaceID, key.PrincipalID); err != nil {
		return database.ProcessSQLErrorf(err, "delete membership query failed")
	}
	return nil
}

// ListForSpace returns a list of memberships for a space.
func (s *MembershipStore) ListForSpace(ctx context.Context, spaceID int64) ([]*types.Membership, error) {
	stmt := database.Builder.
		Select(membershipColumns).
		From("memberships").
		Where("membership_space_id = ?", spaceID).
		OrderBy("membership_created asc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert membership for space list query to sql")
	}

	dst := make([]*membership, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing membership list query")
	}

	result, err := s.mapToMemberships(ctx, dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map memberships to external type: %w", err)
	}

	return result, nil
}

func mapToMembershipNoPrincipalInfo(m *membership) *types.Membership {
	return &types.Membership{
		SpaceID:     m.SpaceID,
		PrincipalID: m.PrincipalID,
		CreatedBy:   m.CreatedBy,
		Created:     m.Created,
		Updated:     m.Updated,
		Role:        m.Role,
	}
}

func mapToInternalMembership(m *types.Membership) *membership {
	return &membership{
		SpaceID:     m.SpaceID,
		PrincipalID: m.PrincipalID,
		CreatedBy:   m.CreatedBy,
		Created:     m.Created,
		Updated:     m.Updated,
		Role:        m.Role,
	}
}

func (s *MembershipStore) mapToMembership(ctx context.Context, m *membership) *types.Membership {
	res := mapToMembershipNoPrincipalInfo(m)

	addedBy, err := s.pCache.Get(ctx, res.CreatedBy)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to load membership creator")
	}
	if addedBy != nil {
		res.AddedBy = *addedBy
	}

	principal, err := s.pCache.Get(ctx, res.PrincipalID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to load membership principal")
	}
	if principal != nil {
		res.Principal = *principal
	}

	return res
}

func (s *MembershipStore) mapToMemberships(ctx context.Context, ms []*membership) ([]*types.Membership, error) {
	// collect all principal IDs
	ids := make([]int64, 0, 2*len(ms))
	for _, m := range ms {
		ids = append(ids, m.CreatedBy, m.PrincipalID)
	}

	// pull principal infos from cache
	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load membership principal infos: %w", err)
	}

	// attach the principal infos back to the slice items
	res := make([]*types.Membership, len(ms))
	for i, m := range ms {
		res[i] = mapToMembershipNoPrincipalInfo(m)
		if addedBy, ok := infoMap[m.CreatedBy]; ok {
			res[i].AddedBy = *addedBy
		}
		if principal, ok := infoMap[m.PrincipalID]; ok {
			res[i].Principal = *principal
		}
	}

	return res, nil
}
