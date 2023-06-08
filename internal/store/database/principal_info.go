// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ store.PrincipalInfoView = (*PrincipalInfoView)(nil)

// NewPrincipalInfoView returns a new PrincipalInfoView.
// It's used by the principal info cache.
func NewPrincipalInfoView(db *sqlx.DB) *PrincipalInfoView {
	return &PrincipalInfoView{
		db: db,
	}
}

type PrincipalInfoView struct {
	db *sqlx.DB
}

const (
	principalInfoCommonColumns = `
		principal_id,
		principal_uid,
		principal_email,
		principal_display_name,
		principal_type,
		principal_created,
		principal_updated`
)

// Find returns a single principal info object by id from the `principals` database table.
func (s *PrincipalInfoView) Find(ctx context.Context, id int64) (*types.PrincipalInfo, error) {
	const sqlQuery = `
		SELECT ` + principalInfoCommonColumns + `
		FROM principals
		WHERE principal_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	v := db.QueryRowContext(ctx, sqlQuery, id)
	if err := v.Err(); err != nil {
		return nil, processSQLErrorf(err, "failed to find principal info")
	}

	info := &types.PrincipalInfo{}

	if err := v.Scan(&info.ID, &info.UID, &info.Email, &info.DisplayName,
		&info.Type, &info.Created, &info.Updated); err != nil {
		return nil, processSQLErrorf(err, "failed to scan principal info")
	}

	return info, nil
}

// FindMany returns a several principal info objects by id from the `principals` database table.
func (s *PrincipalInfoView) FindMany(ctx context.Context, ids []int64) ([]*types.PrincipalInfo, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	stmt := builder.
		Select(principalInfoCommonColumns).
		From("principals").
		Where(squirrel.Eq{"principal_id": ids})

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, processSQLErrorf(err, "failed to generate find many principal info SQL query")
	}

	rows, err := db.QueryContext(ctx, sqlQuery, params...)
	if err != nil {
		return nil, processSQLErrorf(err, "failed to query find many principal info")
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]*types.PrincipalInfo, 0, len(ids))

	for rows.Next() {
		info := &types.PrincipalInfo{}
		err = rows.Scan(&info.ID, &info.UID, &info.Email, &info.DisplayName,
			&info.Type, &info.Created, &info.Updated)
		if err != nil {
			return nil, processSQLErrorf(err, "failed to scan principal info")
		}

		result = append(result, info)
	}

	err = rows.Err()
	if err != nil {
		return nil, processSQLErrorf(err, "failed to read principal info data")
	}

	return result, nil
}
