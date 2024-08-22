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
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type layersDao struct {
	db           *sqlx.DB
	mtRepository store.MediaTypesRepository
}

func NewLayersDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.LayerRepository {
	return &layersDao{
		db:           db,
		mtRepository: mtRepository,
	}
}

type layersDB struct {
	ID          int64 `db:"layer_id"`
	RegistryID  int64 `db:"layer_registry_id"`
	ManifestID  int64 `db:"layer_manifest_id"`
	MediaTypeID int64 `db:"layer_media_type_id"`
	BlobID      int64 `db:"layer_blob_id"`
	Size        int64 `db:"layer_size"`
	CreatedAt   int64 `db:"layer_created_at"`
	UpdatedAt   int64 `db:"layer_updated_at"`
	CreatedBy   int64 `db:"layer_created_by"`
	UpdatedBy   int64 `db:"layer_updated_by"`
}

func (l layersDao) AssociateLayerBlob(ctx context.Context, m *types.Manifest,
	b *types.Blob) error {
	const sqlQuery = `
		INSERT INTO layers ( 
				layer_registry_id
				,layer_manifest_id
				,layer_media_type_id
				,layer_blob_id
				,layer_size
				,layer_created_at
				,layer_updated_at
				,layer_created_by
				,layer_updated_by
		    ) VALUES (
						:layer_registry_id
						,:layer_manifest_id
						,:layer_media_type_id
						,:layer_blob_id
						,:layer_size
						,:layer_created_at
						,:layer_updated_at
						,:layer_created_by
						,:layer_updated_by
		    ) ON CONFLICT (layer_registry_id, layer_manifest_id, layer_blob_id)
			DO NOTHING
			RETURNING layer_id`

	mediaTypeID, err := l.mtRepository.MapMediaType(ctx, b.MediaType)
	if err != nil {
		return err
	}

	layer := &types.Layer{
		RegistryID:  m.RegistryID,
		ManifestID:  m.ID,
		MediaTypeID: mediaTypeID,
		BlobID:      b.ID,
		Size:        b.Size,
	}

	db := dbtx.GetAccessor(ctx, l.db)
	query, arg, err := db.BindNamed(sqlQuery, l.mapToInternalLayer(ctx, layer))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Bind query failed")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&layer.ID); err != nil {
		err = database.ProcessSQLErrorf(ctx, err, "QueryRowContext failed")
		if errors.Is(err, store2.ErrDuplicate) {
			return nil
		}
		if errors.Is(err, store2.ErrForeignKeyViolation) {
			return util.ErrRefManifestNotFound
		}
		return fmt.Errorf("failed to associate layer blob: %w", err)
	}
	return nil
}

func (l layersDao) mapToInternalLayer(ctx context.Context, in *types.Layer) *layersDB {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &layersDB{
		ID:          in.ID,
		RegistryID:  in.RegistryID,
		ManifestID:  in.ManifestID,
		MediaTypeID: in.MediaTypeID,
		BlobID:      in.BlobID,
		Size:        in.Size,
		CreatedAt:   in.CreatedAt.Unix(),
		UpdatedAt:   in.UpdatedAt.Unix(),
		CreatedBy:   in.CreatedBy,
		UpdatedBy:   in.UpdatedBy,
	}
}
