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
	"encoding/json"
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
	"github.com/opencontainers/go-digest"
	errors2 "github.com/pkg/errors"
)

type manifestDao struct {
	sqlDB        *sqlx.DB
	mtRepository store.MediaTypesRepository
}

func NewManifestDao(sqlDB *sqlx.DB, mtRepository store.MediaTypesRepository) store.ManifestRepository {
	return &manifestDao{
		sqlDB:        sqlDB,
		mtRepository: mtRepository,
	}
}

var (
	PrimaryInsertQuery = `
		INSERT INTO manifests ( 
			manifest_registry_id,
			manifest_schema_version,
			manifest_media_type_id,
			manifest_artifact_media_type,
			manifest_total_size,
			manifest_configuration_media_type,
			manifest_configuration_payload,
			manifest_configuration_blob_id,
			manifest_configuration_digest,
			manifest_digest,
			manifest_payload,
			manifest_non_conformant,
			manifest_non_distributable_layers,
			manifest_subject_id,
			manifest_subject_digest,
			manifest_annotations,
			manifest_image_name,
			manifest_created_at,
			manifest_created_by,
			manifest_updated_at,
			manifest_updated_by
		) VALUES (
			:manifest_registry_id,
			:manifest_schema_version,
			:manifest_media_type_id,
			:manifest_artifact_media_type,
			:manifest_total_size,
			:manifest_configuration_media_type,
			:manifest_configuration_payload,
			:manifest_configuration_blob_id,
			:manifest_configuration_digest,
			:manifest_digest,
			:manifest_payload,
			:manifest_non_conformant,
			:manifest_non_distributable_layers,
			:manifest_subject_id,
			:manifest_subject_digest,
			:manifest_annotations,
			:manifest_image_name,
			:manifest_created_at,
			:manifest_created_by,
			:manifest_updated_at,
			:manifest_updated_by
		) RETURNING manifest_id`

	InsertQueryWithConflictHandling = `
		INSERT INTO manifests ( 
			manifest_registry_id,
			manifest_schema_version,
			manifest_media_type_id,
			manifest_artifact_media_type,
			manifest_total_size,
			manifest_configuration_media_type,
			manifest_configuration_payload,
			manifest_configuration_blob_id,
			manifest_configuration_digest,
			manifest_digest,
			manifest_payload,
			manifest_non_conformant,
			manifest_non_distributable_layers,
			manifest_subject_id,
			manifest_subject_digest,
			manifest_annotations,
			manifest_image_name,
			manifest_created_at,
			manifest_created_by,
			manifest_updated_at,
			manifest_updated_by
		) VALUES (
			:manifest_registry_id,
			:manifest_schema_version,
			:manifest_media_type_id,
			:manifest_artifact_media_type,
			:manifest_total_size,
			:manifest_configuration_media_type,
			:manifest_configuration_payload,
			:manifest_configuration_blob_id,
			:manifest_configuration_digest,
			:manifest_digest,
			:manifest_payload,
			:manifest_non_conformant,
			:manifest_non_distributable_layers,
			:manifest_subject_id,
			:manifest_subject_digest,
			:manifest_annotations,
			:manifest_image_name,
			:manifest_created_at,
			:manifest_created_by,
			:manifest_updated_at,
			:manifest_updated_by
		) ON CONFLICT (manifest_registry_id, manifest_image_name, manifest_digest) DO NOTHING
			RETURNING manifest_id`

	ReadQuery = database.Builder.Select(
		"manifest_id", "manifest_registry_id",
		"manifest_total_size", "manifest_schema_version",
		"manifest_media_type_id", "mt_media_type", "manifest_artifact_media_type",
		"manifest_digest", "manifest_payload",
		"manifest_configuration_blob_id", "manifest_configuration_media_type",
		"manifest_configuration_digest",
		"manifest_configuration_payload", "manifest_non_conformant",
		"manifest_non_distributable_layers", "manifest_subject_id",
		"manifest_subject_digest", "manifest_annotations", "manifest_created_at",
		"manifest_created_by", "manifest_updated_at", "manifest_updated_by", "manifest_image_name",
	).
		From("manifests").
		Join("media_types ON mt_id = manifest_media_type_id")
)

// Manifest holds the record of a manifest in DB.
type manifestDB struct {
	ID                     int64          `db:"manifest_id"`
	RegistryID             int64          `db:"manifest_registry_id"`
	TotalSize              int64          `db:"manifest_total_size"`
	SchemaVersion          int            `db:"manifest_schema_version"`
	MediaTypeID            int64          `db:"manifest_media_type_id"`
	ImageName              string         `db:"manifest_image_name"`
	ArtifactMediaType      sql.NullString `db:"manifest_artifact_media_type"`
	Digest                 []byte         `db:"manifest_digest"`
	Payload                []byte         `db:"manifest_payload"`
	ConfigurationMediaType string         `db:"manifest_configuration_media_type"`
	ConfigurationPayload   []byte         `db:"manifest_configuration_payload"`
	ConfigurationDigest    []byte         `db:"manifest_configuration_digest"`
	ConfigurationBlobID    sql.NullInt64  `db:"manifest_configuration_blob_id"`
	SubjectID              sql.NullInt64  `db:"manifest_subject_id"`
	SubjectDigest          []byte         `db:"manifest_subject_digest"`
	NonConformant          bool           `db:"manifest_non_conformant"`
	// NonDistributableLayers identifies whether a manifest
	// references foreign/non-distributable layers. For now, we are
	// not registering metadata about these layers,
	// but we may wish to backfill that metadata in the future by parsing
	// the manifest payload.
	NonDistributableLayers bool   `db:"manifest_non_distributable_layers"`
	Annotations            []byte `db:"manifest_annotations"`
	CreatedAt              int64  `db:"manifest_created_at"`
	CreatedBy              int64  `db:"manifest_created_by"`
	UpdatedAt              int64  `db:"manifest_updated_at"`
	UpdatedBy              int64  `db:"manifest_updated_by"`
}

type manifestMetadataDB struct {
	manifestDB
	MediaType string `db:"mt_media_type"`
}

// FindAll finds all manifests.
func (dao manifestDao) FindAll(_ context.Context) (
	types.Manifests, error,
) {
	// TODO implement me
	panic("implement me")
}

func (dao manifestDao) Count(_ context.Context) (int, error) {
	// TODO implement me
	panic("implement me")
}

func (dao manifestDao) LayerBlobs(
	_ context.Context,
	_ *types.Manifest,
) (types.Blobs, error) {
	// TODO implement me
	panic("implement me")
}

// References finds all manifests directly referenced by a manifest (if any).
func (dao manifestDao) References(
	ctx context.Context,
	m *types.Manifest,
) (types.Manifests, error) {
	stmt := ReadQuery.Join("manifest_references ON manifest_ref_child_id = manifest_id").
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where("manifest_ref_registry_id = ?", m.RegistryID).Where("manifest_ref_parent_id = ?", m.ID)

	db := dbtx.GetAccessor(ctx, dao.sqlDB)
	dst := []*manifestMetadataDB{}

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifests during references")
		return nil, err
	}

	result, err := dao.mapToManifests(dst)
	if err != nil {
		return nil, fmt.Errorf("finding referenced manifests: %w", err)
	}
	return *result, err
}

func (dao manifestDao) Create(ctx context.Context, m *types.Manifest) error {
	mediaTypeID, err := dao.mtRepository.MapMediaType(ctx, m.MediaType)
	if err != nil {
		return fmt.Errorf("mapping manifest media type: %w", err)
	}
	m.MediaTypeID = mediaTypeID

	db := dbtx.GetAccessor(ctx, dao.sqlDB)
	manifest, err := mapToInternalManifest(ctx, m)
	if err != nil {
		return err
	}

	query, arg, err := db.BindNamed(PrimaryInsertQuery, manifest)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind manifest object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&manifest.ID); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Insert query failed")
		if !errors.Is(err, store2.ErrResourceNotFound) {
			return err
		}
	}
	m.ID = manifest.ID
	return nil
}

func (dao manifestDao) CreateOrFind(ctx context.Context, m *types.Manifest) error {
	dgst, err := types.NewDigest(m.Digest)
	if err != nil {
		return err
	}

	mediaTypeID, err := dao.mtRepository.MapMediaType(ctx, m.MediaType)
	if err != nil {
		return fmt.Errorf("mapping manifest media type: %w", err)
	}
	m.MediaTypeID = mediaTypeID

	db := dbtx.GetAccessor(ctx, dao.sqlDB)
	manifest, err := mapToInternalManifest(ctx, m)
	if err != nil {
		return err
	}

	query, arg, err := db.BindNamed(InsertQueryWithConflictHandling, manifest)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind manifest object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&manifest.ID); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Insert query failed")
		if !errors.Is(err, store2.ErrResourceNotFound) {
			return err
		}
		result, err := dao.FindManifestByDigest(ctx, m.RegistryID, m.ImageName, dgst)
		if err != nil {
			return err
		}
		m.ID = result.ID
		return nil
	}

	m.ID = manifest.ID
	return nil
}

func (dao manifestDao) AssociateLayerBlob(
	_ context.Context,
	_ *types.Manifest,
	_ *types.Blob,
) error {
	// TODO implement me
	panic("implement me")
}

func (dao manifestDao) DissociateLayerBlob(
	_ context.Context,
	_ *types.Manifest,
	_ *types.Blob,
) error {
	// TODO implement me
	panic("implement me")
}

func (dao manifestDao) Delete(ctx context.Context, registryID, id int64) error {
	_, err := dao.FindManifestByID(ctx, registryID, id)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get the manifest: %w", err)
	}

	stmt := database.Builder.Delete("manifests").
		Where("manifest_registry_id = ? AND manifest_id = ?", registryID, id)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	_, err = db.ExecContext(ctx, toSQL, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (dao manifestDao) DeleteManifest(
	ctx context.Context, repoID int64,
	imageName string, d digest.Digest,
) (bool, error) {
	digestBytes, err := types.GetDigestBytes(d)
	if err != nil {
		return false, err
	}
	stmt := database.Builder.Delete("manifests").
		Where(
			"manifest_registry_id = ? AND manifest_image_name = ? AND manifest_digest = ?",
			repoID, imageName, digestBytes,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	r, err := db.ExecContext(ctx, toSQL, args...)
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	count, _ := r.RowsAffected()
	return count == 1, nil
}

func (dao manifestDao) DeleteManifestByImageName(
	ctx context.Context, repoID int64,
	imageName string,
) (bool, error) {
	stmt := database.Builder.Delete("manifests").
		Where(
			"manifest_registry_id = ? AND manifest_image_name = ?",
			repoID, imageName,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	r, err := db.ExecContext(ctx, toSQL, args...)
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	count, _ := r.RowsAffected()
	return count > 0, nil
}

func (dao manifestDao) FindManifestByID(
	ctx context.Context,
	registryID,
	id int64,
) (*types.Manifest, error) {
	stmt := database.Builder.Select("manifest_digest").From("manifests").
		Where("manifest_id = ?", id).Where("manifest_registry_id = ?", registryID)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert find manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	return dao.mapToManifest(dst)
}

func (dao manifestDao) FindManifestByDigest(
	ctx context.Context, repoID int64,
	imageName string, digest types.Digest,
) (*types.Manifest, error) {
	digestBytes, err := util.GetHexDecodedBytes(string(digest))
	if err != nil {
		return nil, err
	}

	stmt := ReadQuery.
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where(
			"manifest_registry_id = ? AND manifest_image_name = ? AND manifest_digest = ?",
			repoID, imageName, digestBytes,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	return dao.mapToManifest(dst)
}

func (dao manifestDao) ListManifestsBySubjectDigest(
	ctx context.Context, repoID int64,
	digest types.Digest,
) (types.Manifests, error) {
	digestBytes, err := util.GetHexDecodedBytes(string(digest))
	if err != nil {
		return nil, err
	}

	stmt := ReadQuery.
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where(
			"manifest_registry_id = ? AND manifest_subject_digest = ?",
			repoID, digestBytes,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := []*manifestMetadataDB{}
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.SelectContext(ctx, &dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to list manifests")
		return nil, err
	}

	result, err := dao.mapToManifests(dst)
	if err != nil {
		return nil, fmt.Errorf("finding manifests by subject digest: %w", err)
	}
	return *result, err
}

// FindManifestByTagName finds a manifest by tag name within a repository.
func (dao manifestDao) FindManifestByTagName(
	ctx context.Context, repoID int64,
	imageName string, tag string,
) (*types.Manifest, error) {
	stmt := ReadQuery.
		Join("tags t ON t.tag_registry_id = manifest_registry_id AND t.tag_manifest_id = manifest_id").
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where(
			"manifest_registry_id = ? AND manifest_image_name = ? AND t.tag_name = ?",
			repoID, imageName, tag,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	return dao.mapToManifest(dst)
}

func (dao manifestDao) GetManifestPayload(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
	digest types.Digest,
) (*types.Payload, error) {
	digestBytes, err := util.GetHexDecodedBytes(string(digest))
	if err != nil {
		return nil, err
	}

	stmt := ReadQuery.Join("registries r ON r.registry_id = manifest_registry_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? AND "+
				"manifest_image_name = ? AND manifest_digest = ?",
			parentID, repoKey, imageName, digestBytes,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest payload")
		return nil, err
	}

	m, err := dao.mapToManifest(dst)
	if err != nil {
		return nil, err
	}
	return &m.Payload, nil
}

func (dao manifestDao) FindManifestPayloadByTagName(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
	version string,
) (*types.Payload, error) {
	stmt := ReadQuery.Join("registries r ON r.registry_id = manifest_registry_id").
		Join("tags t ON t.tag_manifest_id = manifest_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ?"+
				" AND manifest_image_name = ? AND t.tag_name = ?",
			parentID, repoKey, imageName, version,
		)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	m, err := dao.mapToManifest(dst)
	if err != nil {
		return nil, err
	}
	return &m.Payload, nil
}

func (dao manifestDao) Get(ctx context.Context, manifestID int64) (*types.Manifest, error) {
	stmt := ReadQuery.
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where("manifest_id = ?", manifestID)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := new(manifestMetadataDB)
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.GetContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	return dao.mapToManifest(dst)
}

func (dao manifestDao) ListManifestsBySubject(
	ctx context.Context,
	repoID int64, id int64,
) (types.Manifests, error) {
	stmt := ReadQuery.
		LeftJoin("blobs ON manifest_configuration_blob_id = blob_id").
		Where("manifest_registry_id = ? AND manifest_subject_id = ?", repoID, id)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest query to sql: %w", err)
	}

	dst := []*manifestMetadataDB{}
	db := dbtx.GetAccessor(ctx, dao.sqlDB)

	if err = db.SelectContext(ctx, dst, toSQL, args...); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Failed to find manifest")
		return nil, err
	}

	result, err := dao.mapToManifests(dst)
	if err != nil {
		return nil, err
	}

	return *result, nil
}

func mapToInternalManifest(ctx context.Context, in *types.Manifest) (*manifestDB, error) {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()

	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	digestBytes, err := types.GetDigestBytes(in.Digest)
	if err != nil {
		return nil, err
	}

	var configBlobID sql.NullInt64
	var configPayload types.Payload
	var configMediaType string
	var cfgDigestBytes []byte
	if in.Configuration != nil {
		configPayload = in.Configuration.Payload
		configMediaType = in.Configuration.MediaType
		configBlobID = sql.NullInt64{Int64: in.Configuration.BlobID, Valid: true}

		cfgDigestBytes, err = types.GetDigestBytes(in.Configuration.Digest)
		if err != nil {
			return nil, err
		}
	}

	sbjDigestBytes, err := types.GetDigestBytes(in.SubjectDigest)
	if err != nil {
		return nil, err
	}

	annot, err := json.Marshal(in.Annotations)
	if err != nil {
		return nil, err
	}

	return &manifestDB{
		ID:                     in.ID,
		RegistryID:             in.RegistryID,
		TotalSize:              in.TotalSize,
		SchemaVersion:          in.SchemaVersion,
		MediaTypeID:            in.MediaTypeID,
		ArtifactMediaType:      in.ArtifactType,
		Digest:                 digestBytes,
		Payload:                in.Payload,
		ConfigurationBlobID:    configBlobID,
		ConfigurationMediaType: configMediaType,
		ConfigurationPayload:   configPayload,
		ConfigurationDigest:    cfgDigestBytes,
		NonConformant:          in.NonConformant,
		NonDistributableLayers: in.NonDistributableLayers,
		SubjectID:              in.SubjectID,
		SubjectDigest:          sbjDigestBytes,
		Annotations:            annot,
		ImageName:              in.ImageName,
		CreatedAt:              in.CreatedAt.UnixMilli(),
		CreatedBy:              in.CreatedBy,
		UpdatedBy:              in.UpdatedBy,
	}, nil
}

func (dao manifestDao) mapToManifest(dst *manifestMetadataDB) (*types.Manifest, error) {
	// Converting []byte digest into Digest
	dgst := types.Digest(util.GetHexEncodedString(dst.Digest))
	parsedDigest, err := dgst.Parse()
	if err != nil {
		return nil, err
	}

	// Converting Configuration []byte digest into Digest
	cfgDigest := types.Digest(util.GetHexEncodedString(dst.ConfigurationDigest))
	cfgParsedDigest, err := cfgDigest.Parse()
	if err != nil {
		return nil, err
	}

	// Converting Subject []byte digest into Digest
	sbjDigest := types.Digest(util.GetHexEncodedString(dst.SubjectDigest))
	sbjParsedDigest, err := sbjDigest.Parse()
	if err != nil {
		return nil, err
	}

	var annot map[string]string
	err = json.Unmarshal(dst.Annotations, &annot)
	if err != nil {
		return nil, err
	}

	m := &types.Manifest{
		ID:                     dst.ID,
		RegistryID:             dst.RegistryID,
		TotalSize:              dst.TotalSize,
		SchemaVersion:          dst.SchemaVersion,
		MediaTypeID:            dst.MediaTypeID,
		MediaType:              dst.MediaType,
		ArtifactType:           dst.ArtifactMediaType,
		Digest:                 parsedDigest,
		Payload:                dst.Payload,
		NonConformant:          dst.NonConformant,
		NonDistributableLayers: dst.NonDistributableLayers,
		SubjectID:              dst.SubjectID,
		SubjectDigest:          sbjParsedDigest,
		Annotations:            annot,
		ImageName:              dst.ImageName,
		CreatedAt:              time.UnixMilli(dst.CreatedAt),
	}

	if dst.ConfigurationBlobID.Valid {
		m.Configuration = &types.Configuration{
			BlobID:    dst.ConfigurationBlobID.Int64,
			MediaType: dst.ConfigurationMediaType,
			Digest:    cfgParsedDigest,
			Payload:   dst.ConfigurationPayload,
		}
	}

	return m, nil
}

func (dao manifestDao) mapToManifests(dst []*manifestMetadataDB) (*types.Manifests, error) {
	mm := make(types.Manifests, 0, len(dst))

	for _, d := range dst {
		m, err := dao.mapToManifest(d)
		if err != nil {
			return nil, err
		}
		mm = append(mm, m)
	}

	return &mm, nil
}
