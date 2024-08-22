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
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type manifestReferenceDao struct {
	db *sqlx.DB
}

func NewManifestReferenceDao(db *sqlx.DB) store.ManifestReferenceRepository {
	return &manifestReferenceDao{
		db: db,
	}
}

type manifestReferenceDB struct {
	ID         int64 `db:"manifest_ref_id"`
	RegistryID int64 `db:"manifest_ref_registry_id"`
	ParentID   int64 `db:"manifest_ref_parent_id"`
	ChildID    int64 `db:"manifest_ref_child_id"`
	CreatedAt  int64 `db:"manifest_ref_created_at"`
	UpdatedAt  int64 `db:"manifest_ref_updated_at"`
	CreatedBy  int64 `db:"manifest_ref_created_by"`
	UpdatedBy  int64 `db:"manifest_ref_updated_by"`
}

func (dao manifestReferenceDao) AssociateManifest(
	ctx context.Context,
	ml *types.Manifest, m *types.Manifest,
) error {
	if ml.ID == m.ID {
		return fmt.Errorf("cannot associate a manifest with itself")
	}
	const sqlQuery = `
		INSERT INTO manifest_references ( 
			manifest_ref_registry_id
			,manifest_ref_parent_id
			,manifest_ref_child_id
			,manifest_ref_created_at
			,manifest_ref_updated_at
			,manifest_ref_created_by
			,manifest_ref_updated_by
		) VALUES (
			:manifest_ref_registry_id
			,:manifest_ref_parent_id
			,:manifest_ref_child_id
			,:manifest_ref_created_at
			,:manifest_ref_updated_at
			,:manifest_ref_created_by
			,:manifest_ref_updated_by
		) ON CONFLICT (manifest_ref_registry_id, manifest_ref_parent_id, manifest_ref_child_id)
			DO NOTHING
			RETURNING manifest_ref_id`

	manifestRef := &types.ManifestReference{
		RegistryID: ml.RegistryID,
		ParentID:   ml.ID,
		ChildID:    m.ID,
	}

	db := dbtx.GetAccessor(ctx, dao.db)
	query, arg, err := db.BindNamed(sqlQuery, mapToInternalManifestReference(ctx, manifestRef))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Bind query failed")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&manifestRef.ID); err != nil {
		err = databaseg.ProcessSQLErrorf(ctx, err, "QueryRowContext failed")
		if errors.Is(err, store2.ErrDuplicate) {
			return nil
		}
		if errors.Is(err, store2.ErrForeignKeyViolation) {
			return util.ErrRefManifestNotFound
		}
		return fmt.Errorf("inserting manifest reference: %w", err)
	}
	return nil
}

func (dao manifestReferenceDao) DissociateManifest(
	_ context.Context,
	_ *types.Manifest,
	_ *types.Manifest,
) error {
	// TODO implement me
	panic("implement me")
}

func mapToInternalManifestReference(ctx context.Context, in *types.ManifestReference) *manifestReferenceDB {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &manifestReferenceDB{
		ID:         in.ID,
		RegistryID: in.RegistryID,
		ParentID:   in.ParentID,
		ChildID:    in.ChildID,
		CreatedAt:  in.CreatedAt.UnixMilli(),
		UpdatedAt:  in.UpdatedAt.UnixMilli(),
		CreatedBy:  in.CreatedBy,
		UpdatedBy:  in.UpdatedBy,
	}
}
