//  Copyright 2023 Harness, Inc.
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
	"database/sql"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/types/enum"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type CleanupPolicyDao struct {
	db *sqlx.DB
	tx dbtx.Transactor
}

type CleanupPolicyDB struct {
	ID             int64  `db:"cp_id"`
	RegistryID     int64  `db:"cp_registry_id"`
	Name           string `db:"cp_name"`
	ExpiryTimeInMs int64  `db:"cp_expiry_time_ms"`
	CreatedAt      int64  `db:"cp_created_at"`
	UpdatedAt      int64  `db:"cp_updated_at"`
	CreatedBy      int64  `db:"cp_created_by"`
	UpdatedBy      int64  `db:"cp_updated_by"`
}

type CleanupPolicyPrefixMappingDB struct {
	PrefixID        int64           `db:"cpp_id"`
	CleanupPolicyID int64           `db:"cpp_cleanup_policy_id"`
	Prefix          string          `db:"cpp_prefix"`
	PrefixType      enum.PrefixType `db:"cpp_prefix_type"`
}

type CleanupPolicyJoinMapping struct {
	CleanupPolicyDB
	CleanupPolicyPrefixMappingDB
}

func NewCleanupPolicyDao(db *sqlx.DB, tx dbtx.Transactor) store.CleanupPolicyRepository {
	return &CleanupPolicyDao{
		db: db,
		tx: tx,
	}
}

func (c CleanupPolicyDao) GetIDsByRegistryID(ctx context.Context, id int64) (ids []int64, err error) {
	stmt := databaseg.Builder.Select("cp_id").From("cleanup_policies").
		Where("cp_registry_id = ?", id)
	db := dbtx.GetAccessor(ctx, c.db)
	var res []int64
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, err
	}
	if err = db.SelectContext(ctx, &res, query, args...); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, databaseg.ProcessSQLErrorf(
				ctx, err,
				"failed to get cleanup policy ids by registry id %d", id,
			)
		}
	}

	return res, nil
}

func (c CleanupPolicyDao) GetByRegistryID(
	ctx context.Context,
	id int64,
) (cleanupPolicies *[]types.CleanupPolicy, err error) {
	stmt := databaseg.Builder.Select(
		"cp_id",
		"cp_registry_id",
		"cp_name",
		"cp_expiry_time_ms",
		"cp_created_at",
		"cp_updated_at",
		"cp_created_by",
		"cp_updated_by",
		"cpp_id",
		"cpp_cleanup_policy_id",
		"cpp_prefix",
		"cpp_prefix_type",
	).
		From("cleanup_policies").
		Join("cleanup_policy_prefix_mappings ON cp_id = cpp_cleanup_policy_id").
		Where("cp_registry_id = ?", id)

	db := dbtx.GetAccessor(ctx, c.db)
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, databaseg.ProcessSQLErrorf(
				ctx, err,
				"failed to get cleanup policy ids by registry id %d", id,
			)
		}
	}

	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to close rows: %v", err)
		}
	}(rows)

	return c.mapToCleanupPolicies(ctx, rows)
}

func (c CleanupPolicyDao) Create(ctx context.Context, cleanupPolicy *types.CleanupPolicy) (id int64, err error) {
	const sqlQuery = `
		INSERT INTO cleanup_policies (
			cp_registry_id
			,cp_name
			,cp_expiry_time_ms
			,cp_created_at
			,cp_updated_at
			,cp_created_by
			,cp_updated_by
		) values (
			:cp_registry_id
			,:cp_name
			,:cp_expiry_time_ms
			,:cp_created_at
			,:cp_updated_at
			,:cp_created_by
			,:cp_updated_by
		) RETURNING cp_id`

	db := dbtx.GetAccessor(ctx, c.db)

	// insert repo first so we get id
	query, arg, err := db.BindNamed(sqlQuery, c.mapToInternalCleanupPolicy(ctx, cleanupPolicy))
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(
			ctx,
			err, "Failed to bind cleanup policy object",
		)
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&cleanupPolicy.ID); err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return cleanupPolicy.ID, nil
}

func (c CleanupPolicyDao) createPrefixMapping(
	ctx context.Context,
	mapping CleanupPolicyPrefixMappingDB,
) (id int64, err error) {
	const sqlQuery = `
		INSERT INTO cleanup_policy_prefix_mappings (
			cpp_cleanup_policy_id
			,cpp_prefix
			,cpp_prefix_type
		) values (
			:cpp_cleanup_policy_id
			,:cpp_prefix
			,:cpp_prefix_type
		) RETURNING cpp_id`

	db := dbtx.GetAccessor(ctx, c.db)

	// insert repo first so we get id
	query, arg, err := db.BindNamed(sqlQuery, mapping)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(
			ctx, err,
			"Failed to bind cleanup policy object",
		)
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&mapping.PrefixID); err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return mapping.PrefixID, nil
}

// Delete deletes a cleanup policy by id
// It doesn't cleanup the prefix mapping as they are removed by the database cascade.
func (c CleanupPolicyDao) Delete(ctx context.Context, id int64) (err error) {
	stmt := databaseg.Builder.Delete("cleanup_policies").Where("cp_id = ?", id)
	query, args, err := stmt.ToSql()
	if err != nil {
		return err
	}

	db := dbtx.GetAccessor(ctx, c.db)
	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(
			ctx, err,
			"failed to delete cleanup policy %d", id,
		)
	}
	return nil
}

func (c CleanupPolicyDao) deleteCleanupPolicies(ctx context.Context, ids []int64) error {
	query, args, err := sqlx.In("DELETE FROM cleanup_policies WHERE cp_id IN (?)", ids)
	if err != nil {
		return err
	}

	query = c.db.Rebind(query)
	db := dbtx.GetAccessor(ctx, c.db)
	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(
			ctx, err,
			"failed to delete cleanup policies %v", ids,
		)
	}
	return nil
}

func (c CleanupPolicyDao) ModifyCleanupPolicies(
	ctx context.Context,
	cleanupPolicies *[]types.CleanupPolicy,
	ids []int64,
) error {
	err := c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			if len(ids) > 0 {
				err := c.deleteCleanupPolicies(ctx, ids)
				if err != nil {
					return err
				}
			}

			if !commons.IsEmpty(cleanupPolicies) {
				for _, cp := range *cleanupPolicies {
					cpCopy := cp // Create a copy of cp to avoid implicit memory aliasing
					id, err := c.Create(ctx, &cpCopy)
					if err != nil {
						return err
					}

					cp.ID = id
					err2 := c.createPrefixMappingsInternal(ctx, cp)
					if err2 != nil {
						return err2
					}
				}
			}
			return nil
		},
	)
	return err
}

func (c CleanupPolicyDao) createPrefixMappingsInternal(
	ctx context.Context,
	cp types.CleanupPolicy,
) error {
	mappings := c.mapToInternalCleanupPolicyMapping(&cp)
	for _, m := range *mappings {
		_, err := c.createPrefixMapping(ctx, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c CleanupPolicyDao) mapToInternalCleanupPolicyMapping(
	cp *types.CleanupPolicy,
) *[]CleanupPolicyPrefixMappingDB {
	result := make([]CleanupPolicyPrefixMappingDB, 0)
	if !commons.IsEmpty(cp.PackagePrefix) {
		for _, prefix := range cp.PackagePrefix {
			result = append(
				result, CleanupPolicyPrefixMappingDB{
					CleanupPolicyID: cp.ID,
					Prefix:          prefix,
					PrefixType:      enum.PrefixTypePackage,
				},
			)
		}
	}
	if !commons.IsEmpty(cp.VersionPrefix) {
		for _, prefix := range cp.VersionPrefix {
			result = append(
				result, CleanupPolicyPrefixMappingDB{
					CleanupPolicyID: cp.ID,
					Prefix:          prefix,
					PrefixType:      enum.PrefixTypeVersion,
				},
			)
		}
	}
	return &result
}

func (c CleanupPolicyDao) mapToInternalCleanupPolicy(
	ctx context.Context,
	cp *types.CleanupPolicy,
) *CleanupPolicyDB {
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}

	cp.UpdatedAt = time.Now()

	session, _ := request.AuthSessionFrom(ctx)
	if cp.CreatedBy == 0 {
		cp.CreatedBy = session.Principal.ID
	}
	cp.UpdatedBy = session.Principal.ID

	return &CleanupPolicyDB{
		ID:             cp.ID,
		RegistryID:     cp.RegistryID,
		Name:           cp.Name,
		ExpiryTimeInMs: cp.ExpiryTime,
		CreatedAt:      cp.CreatedAt.UnixMilli(),
		UpdatedAt:      cp.UpdatedAt.UnixMilli(),
		CreatedBy:      cp.CreatedBy,
		UpdatedBy:      cp.UpdatedBy,
	}
}

func (c CleanupPolicyDao) mapToCleanupPolicies(
	_ context.Context,
	rows *sqlx.Rows,
) (*[]types.CleanupPolicy, error) {
	cleanupPolicies := make(map[int64]*types.CleanupPolicy)

	for rows.Next() {
		var cp CleanupPolicyJoinMapping
		if err := rows.StructScan(&cp); err != nil {
			return nil,
				errors.Wrap(err, "failed to scan cleanup policy")
		}

		if _, exists := cleanupPolicies[cp.ID]; !exists {
			cleanupPolicies[cp.ID] = &types.CleanupPolicy{
				ID:            cp.ID,
				RegistryID:    cp.RegistryID,
				Name:          cp.Name,
				ExpiryTime:    cp.ExpiryTimeInMs,
				CreatedAt:     time.UnixMilli(cp.CreatedAt),
				UpdatedAt:     time.UnixMilli(cp.UpdatedAt),
				PackagePrefix: make([]string, 0),
				VersionPrefix: make([]string, 0),
			}
		}

		if cp.PrefixType == enum.PrefixTypePackage {
			cleanupPolicies[cp.ID].PackagePrefix = append(cleanupPolicies[cp.ID].PackagePrefix, cp.Prefix)
		}

		if cp.PrefixType == enum.PrefixTypeVersion {
			cleanupPolicies[cp.ID].VersionPrefix = append(cleanupPolicies[cp.ID].VersionPrefix, cp.Prefix)
		}
	}
	var result []types.CleanupPolicy
	for _, cp := range cleanupPolicies {
		result = append(result, *cp)
	}
	return &result, nil
}
