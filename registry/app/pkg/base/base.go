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

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	registryaudit "github.com/harness/gitness/registry/app/pkg/audit"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	registryrefcache "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ LocalBase = (*localBase)(nil)

type LocalBase interface {

	// UploadFile uploads the file to the storage.
	// FIXME: Validate upload by given sha256 or any other checksums provided
	UploadFile(
		ctx context.Context,
		info pkg.ArtifactInfo,
		fileName string,
		version string,
		path string,
		file multipart.File,
		metadata metadata.Metadata,
	) (headers *commons.ResponseHeaders, sha256 string, err error)
	Upload(
		ctx context.Context,
		info pkg.ArtifactInfo,
		fileName,
		version,
		path string,
		file io.ReadCloser,
		metadata metadata.Metadata,
	) (*commons.ResponseHeaders, string, error)
	UpdateFileManagerAndCreateArtifact(
		ctx context.Context,
		info pkg.ArtifactInfo,
		version, path string,
		metadata metadata.Metadata,
		fileInfo types.FileInfo,
		failOnConflict bool,
	) (*commons.ResponseHeaders, string, int64, bool, error)
	Download(ctx context.Context, info pkg.ArtifactInfo, version string, fileName string) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)

	Exists(ctx context.Context, info pkg.ArtifactInfo, path string) bool
	ExistsE(ctx context.Context, info pkg.PackageArtifactInfo, path string) (
		headers *commons.ResponseHeaders,
		err error,
	)
	DeleteFile(ctx context.Context, info pkg.PackageArtifactInfo, filePath string) (
		headers *commons.ResponseHeaders,
		err error,
	)

	ExistsByFilePath(ctx context.Context, registryID int64, filePath string) (bool, error)

	// AuditPush logs audit trail for artifact push operations.
	AuditPush(
		ctx context.Context, info pkg.ArtifactInfo, version string,
		imageUUID string, artifactUUID string,
	)

	CheckIfVersionExists(ctx context.Context, info pkg.PackageArtifactInfo) (bool, error)

	DeletePackage(ctx context.Context, info pkg.PackageArtifactInfo) error

	DeleteVersion(ctx context.Context, info pkg.PackageArtifactInfo) error

	MoveMultipleTempFilesAndCreateArtifact(
		ctx context.Context,
		info *pkg.ArtifactInfo,
		pathPrefix string,
		metadata metadata.Metadata,
		filesInfo *[]types.FileInfo,
		version string,
	) error
}

type localBase struct {
	registryDao    store.RegistryRepository
	registryFinder registryrefcache.RegistryFinder
	fileManager    filemanager.FileManager
	tx             dbtx.Transactor
	imageDao       store.ImageRepository
	artifactDao    store.ArtifactRepository
	nodesDao       store.NodesRepository
	tagsDao        store.PackageTagRepository
	authorizer     authz.Authorizer
	spaceFinder    refcache.SpaceFinder
	auditService   audit.Service
}

func NewLocalBase(
	registryDao store.RegistryRepository,
	registryFinder registryrefcache.RegistryFinder,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	nodesDao store.NodesRepository,
	tagsDao store.PackageTagRepository,
	authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder,
	auditService audit.Service,
) LocalBase {
	return &localBase{
		registryDao:    registryDao,
		registryFinder: registryFinder,
		fileManager:    fileManager,
		tx:             tx,
		imageDao:       imageDao,
		artifactDao:    artifactDao,
		nodesDao:       nodesDao,
		tagsDao:        tagsDao,
		authorizer:     authorizer,
		spaceFinder:    spaceFinder,
		auditService:   auditService,
	}
}

func (l *localBase) UploadFile(
	ctx context.Context,
	info pkg.ArtifactInfo,
	fileName string,
	version string,
	path string,
	file multipart.File,
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	return l.uploadInternal(ctx, info, fileName, version, path, file, nil, metadata)
}

func (l *localBase) Upload(
	ctx context.Context,
	info pkg.ArtifactInfo,
	fileName,
	version,
	path string,
	file io.ReadCloser,
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	return l.uploadInternal(ctx, info, fileName, version, path, nil, file, metadata)
}

func (l *localBase) UpdateFileManagerAndCreateArtifact(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version, path string,
	metadata metadata.Metadata,
	fileInfo types.FileInfo,
	failOnConflict bool,
) (response *commons.ResponseHeaders, sha256 string, artifactID int64, isExistent bool, err error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	err = l.CheckIfFileAlreadyExist(ctx, info, version, metadata, fileInfo.Filename, path)
	if err != nil {
		if !errors.IsConflict(err) {
			return nil, "", 0, false, err
		}
		if failOnConflict {
			responseHeaders.Code = http.StatusConflict
			return responseHeaders, "", 0, true,
				usererror.Conflict(fmt.Sprintf("File with name:[%s],"+
					" package:[%s], version:[%s] already exist", fileInfo.Filename, info.Image, version))
		}
		_, fileSha256, err2 := l.GetSHA256ByPath(ctx, info.RegistryID, path)
		if err2 != nil {
			return responseHeaders, "", 0, true, err2
		}

		responseHeaders.Code = http.StatusCreated
		return responseHeaders, fileSha256, 0, true, nil
	}

	registry, err := l.registryFinder.FindByRootParentID(ctx, info.RootParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, "", 0, false, errcode.ErrCodeUnknown.WithDetail(err)
	}
	session, _ := request.AuthSessionFrom(ctx)
	err = l.fileManager.PostFileUpload(ctx, path, registry.ID, info.RootParentID,
		info.RootIdentifier, fileInfo, session.Principal.ID)
	if err != nil {
		return responseHeaders, "", 0, false, errcode.ErrCodeUnknown.WithDetail(err)
	}

	artifactID, err = l.postUploadArtifact(ctx, info, registry, version, metadata, fileInfo)
	if err != nil {
		return responseHeaders, "", 0, false, err
	}
	responseHeaders.Code = http.StatusCreated
	return responseHeaders, fileInfo.Sha256, artifactID, false, nil
}

func (l *localBase) MoveMultipleTempFilesAndCreateArtifact(
	ctx context.Context,
	info *pkg.ArtifactInfo,
	pathPrefix string,
	metadata metadata.Metadata,
	filesInfo *[]types.FileInfo,
	version string,
) error {
	session, _ := request.AuthSessionFrom(ctx)
	for _, fileInfo := range *filesInfo {
		filePath := path.Join(pathPrefix, fileInfo.Filename)
		_, err := l.fileManager.GetFilePath(ctx, fileInfo.Sha256, info.RegistryID, info.RootParentID)
		if err == nil {
			// It means file already exist, or it was moved in some iteration, no need to move it, just save nodes
			err = l.fileManager.SaveNodes(ctx, fileInfo.Sha256, info.RegistryID, info.RootParentID,
				session.Principal.ID,
				fileInfo.Sha256)
			if err != nil {
				log.Ctx(ctx).Info().Msgf("Failed to move filesInfo with sha %s to %s", fileInfo.Sha256,
					fileInfo.Filename)
				return err
			}
			continue
		}
		// Otherwise, move the file to permanent location and save nodes
		err = l.fileManager.PostFileUpload(ctx, filePath, info.RegistryID, info.RootParentID,
			info.RootIdentifier, fileInfo, session.Principal.ID)
		if err != nil {
			log.Ctx(ctx).Info().Msgf("Failed to move filesInfo with sha %s to %s", fileInfo.Sha256,
				fileInfo.Filename)
			return err
		}
	}

	var imageUUID string
	var artifactUUID string
	err := l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			image := &types.Image{
				Name:         info.Image,
				RegistryID:   info.RegistryID,
				ArtifactType: info.ArtifactType,
				Enabled:      true,
			}
			// Create or update image
			err := l.imageDao.CreateOrUpdate(ctx, image)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to create image for artifact: [%s] with error: %v",
					info.Image, err)
				return fmt.Errorf("failed to create image for artifact: [%s], error: %w",
					info.Image, err)
			}
			imageUUID = image.UUID

			dbArtifact, err := l.artifactDao.GetByName(ctx, image.ID, version)

			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				log.Ctx(ctx).Error().Msgf("Failed to fetch artifact : [%s] with error: %v", version, err)
				return fmt.Errorf("failed to fetch artifact : [%s] with error: %w", info.Image, err)
			}

			// Update metadata
			err2 := l.updateFilesMetadata(ctx, dbArtifact, metadata, info, filesInfo)
			if err2 != nil {
				log.Ctx(ctx).Error().Msgf("Failed to update metadata for artifact: [%s] with error: %v",
					info.Image, err2)
				return fmt.Errorf("failed to update metadata for artifact: [%s] with error: %w", info.Image, err2)
			}

			metadataJSON, err := json.Marshal(metadata)

			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to parse metadata for artifact: [%s] with error: %v",
					info.Image, err)
				return fmt.Errorf("failed to parse metadata for artifact: [%s] with error: %w", info.Image, err)
			}

			// Create or update artifact
			newArtifact := &types.Artifact{
				ImageID:  image.ID,
				Version:  version,
				Metadata: metadataJSON,
			}

			_, err = l.artifactDao.CreateOrUpdate(ctx, newArtifact)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("Failed to create artifact : [%s] with error: %v", info.Image, err)
				return fmt.Errorf("failed to create artifact : [%s] with error: %w", info.Image, err)
			}

			// UUID is populated by CreateOrUpdate (generated in mapToInternalArtifact)
			artifactUUID = newArtifact.UUID

			return nil
		})
	if err != nil {
		return err
	}

	// Audit log for artifact push
	l.AuditPush(ctx, *info, version, imageUUID, artifactUUID)

	return nil
}

func (l *localBase) updateFilesMetadata(
	ctx context.Context,
	dbArtifact *types.Artifact,
	inputMetadata metadata.Metadata,
	info *pkg.ArtifactInfo,
	filesInfo *[]types.FileInfo,
) error {
	var files []metadata.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, inputMetadata)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("Failed to get metadata for artifact: [%s] with registry: [%s] and"+
				" error: %v", info.Image, info.RegIdentifier, err)
			return fmt.Errorf("failed to get metadata for artifact: [%s] with registry: [%s] and error: %w",
				info.Image, info.RegIdentifier, err)
		}
	}
	for _, fileInfo := range *filesInfo {
		fileExist := false
		files = inputMetadata.GetFiles()
		for _, file := range files {
			if file.Filename == fileInfo.Filename {
				fileExist = true
			}
		}
		if !fileExist {
			files = append(files, metadata.File{
				Size: fileInfo.Size, Filename: fileInfo.Filename,
				CreatedAt: time.Now().UnixMilli(),
				Sha256:    fileInfo.Sha256,
			})
			inputMetadata.SetFiles(files)
			inputMetadata.UpdateSize(fileInfo.Size)
		}
	}
	return nil
}

func (l *localBase) uploadInternal(
	ctx context.Context,
	info pkg.ArtifactInfo,
	fileName string,
	version string,
	path string,
	file multipart.File,
	fileReadCloser io.ReadCloser,
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	err := l.CheckIfFileAlreadyExist(ctx, info, version, metadata, fileName, path)

	if err != nil {
		if !errors.IsConflict(err) {
			return nil, "", err
		}
		err = pkg.GetRegistryCheckAccess(ctx, l.authorizer, l.spaceFinder,
			info.ParentID, info, enum.PermissionArtifactsDelete)
		if err != nil {
			return nil, "", usererror.Forbidden(fmt.Sprintf("Not enough permissions to overwrite file %s "+
				"(needs DELETE permission).",
				fileName))
		}
	}

	registry, err := l.registryFinder.FindByRootParentID(ctx, info.RootParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	session, _ := request.AuthSessionFrom(ctx)
	fileInfo, err := l.fileManager.UploadFile(ctx, path, registry.ID, info.RootParentID, info.RootIdentifier, file,
		fileReadCloser, session.Principal.ID)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	fileInfo.Filename = fileName
	_, err = l.postUploadArtifact(ctx, info, registry, version, metadata, fileInfo)
	if err != nil {
		return responseHeaders, "", err
	}
	responseHeaders.Code = http.StatusCreated
	return responseHeaders, fileInfo.Sha256, nil
}

func (l *localBase) postUploadArtifact(
	ctx context.Context,
	info pkg.ArtifactInfo,
	registry *types.Registry,
	version string,
	metadata metadata.Metadata,
	fileInfo types.FileInfo,
) (int64, error) {
	var artifactID int64
	var imageUUID string
	var artifactUUID string
	err := l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			image := &types.Image{
				Name:       info.Image,
				RegistryID: registry.ID,
				Enabled:    true,
			}
			err := l.imageDao.CreateOrUpdate(ctx, image)
			if err != nil {
				return fmt.Errorf("failed to create image for artifact: [%s], error: %w", info.Image, err)
			}

			imageUUID = image.UUID

			dbArtifact, err := l.artifactDao.GetByName(ctx, image.ID, version)

			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				return fmt.Errorf("failed to fetch artifact : [%s] with error: %w", info.Image, err)
			}

			metadataJSON := []byte("{}")
			if metadata != nil {
				err2 := l.updateMetadata(dbArtifact, metadata, info, fileInfo)
				if err2 != nil {
					return fmt.Errorf("failed to update metadata for artifact: [%s] with error: %w", info.Image, err2)
				}

				metadataJSON, err = json.Marshal(metadata)

				if err != nil {
					return fmt.Errorf("failed to parse metadata for artifact: [%s] with error: %w", info.Image, err)
				}
			}

			newArtifact := &types.Artifact{
				ImageID:  image.ID,
				Version:  version,
				Metadata: metadataJSON,
			}

			artifactID, err = l.artifactDao.CreateOrUpdate(ctx, newArtifact)
			if err != nil {
				return fmt.Errorf("failed to create artifact : [%s] with error: %w", info.Image, err)
			}

			// Audit log for push/upload operation
			// UUID is populated by CreateOrUpdate (generated in mapToInternalArtifact)
			artifactUUID = newArtifact.UUID
			l.AuditPush(ctx, info, version, imageUUID, artifactUUID)

			return nil
		})
	return artifactID, err
}

func (l *localBase) Download(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version string,
	fileName string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	path := "/" + info.Image + "/" + version + "/" + fileName
	reg, _ := l.registryFinder.FindByRootParentID(ctx, info.RootParentID, info.RegIdentifier)

	fileReader, _, redirectURL, err := l.fileManager.DownloadFileByPath(ctx, path, reg.ID,
		info.RegIdentifier, info.RootIdentifier, true)
	if err != nil {
		return responseHeaders, nil, "", err
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, nil
}

func (l *localBase) Exists(ctx context.Context, info pkg.ArtifactInfo, path string) bool {
	exists, _, _, err := l.getSHA256(ctx, info.RegistryID, path)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("Failed to check if file: [%s] exists", path)
	}
	return exists
}

func (l *localBase) ExistsE(
	ctx context.Context,
	info pkg.PackageArtifactInfo,
	filePath string,
) (headers *commons.ResponseHeaders, err error) {
	exists, sha256, size, err := l.getSHA256(ctx, info.GetRegistryID(), GetCompletePath(info, filePath))
	headers = &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	// TODO: Need better error handling
	if exists {
		headers.Code = http.StatusOK
		headers.Headers[storage.HeaderContentDigest] = sha256
		headers.Headers[storage.HeaderContentLength] = strconv.FormatInt(size, 10)
		headers.Headers[storage.HeaderEtag] = "sha256:" + sha256
	} else {
		headers.Code = http.StatusNotFound
	}
	return headers, err
}

func (l *localBase) DeleteFile(ctx context.Context, info pkg.PackageArtifactInfo, filePath string) (
	headers *commons.ResponseHeaders,
	err error,
) {
	completePath := GetCompletePath(info, filePath)
	exists, _, _, _ := l.getSHA256(ctx, info.GetRegistryID(), completePath)
	if exists {
		err = l.fileManager.DeleteLeafNode(ctx, info.GetRegistryID(), completePath)
		if err != nil {
			log.Ctx(ctx).Error().Stack().Err(err).Msgf("Failed to delete file: %q, registry: %d", completePath,
				info.GetRegistryID())
			return nil, err
		}
		return &commons.ResponseHeaders{
			Code:    204,
			Headers: map[string]string{},
		}, nil
	}
	return &commons.ResponseHeaders{
		Code:    http.StatusNotFound,
		Headers: map[string]string{},
	}, nil
}

func (l *localBase) ExistsByFilePath(ctx context.Context, registryID int64, filePath string) (bool, error) {
	exists, _, err := l.GetSHA256ByPath(ctx, registryID, filePath)
	if err != nil && strings.Contains(err.Error(), "resource not found") {
		return false, nil
	}
	return exists, err
}

func (l *localBase) CheckIfVersionExists(ctx context.Context, info pkg.PackageArtifactInfo) (bool, error) {
	_, err := l.artifactDao.GetByRegistryImageAndVersion(ctx,
		info.BaseArtifactInfo().RegistryID, info.BaseArtifactInfo().Image, info.GetVersion())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (l *localBase) getSHA256(ctx context.Context, registryID int64, path string) (
	exists bool,
	sha256 string,
	size int64,
	err error,
) {
	filePath := "/" + path
	sha256, size, err = l.fileManager.HeadFile(ctx, filePath, registryID)
	if err != nil {
		return false, "", 0, err
	}

	//FIXME: err should be checked on if the record doesn't exist or there was DB call issue
	return true, sha256, size, nil
}

func (l *localBase) GetSHA256ByPath(ctx context.Context, registryID int64, filePath string) (
	exists bool,
	sha256 string,
	err error,
) {
	sha256, _, err = l.fileManager.HeadFile(ctx, "/"+filePath, registryID)
	if err != nil {
		return false, "", err
	}

	//FIXME: err should be checked on if the record doesn't exist or there was DB call issue
	return true, sha256, err
}

func (l *localBase) updateMetadata(
	dbArtifact *types.Artifact,
	inputMetadata metadata.Metadata,
	info pkg.ArtifactInfo,
	fileInfo types.FileInfo,
) error {
	var files []metadata.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, inputMetadata)
		if err != nil {
			return fmt.Errorf("failed to get metadata for artifact: [%s] with registry: [%s] and error: %w", info.Image,
				info.RegIdentifier, err)
		}
		fileExist := false
		files = inputMetadata.GetFiles()
		for _, file := range files {
			if file.Filename == fileInfo.Filename {
				fileExist = true
			}
		}
		if !fileExist {
			files = append(files, metadata.File{
				Size: fileInfo.Size, Filename: fileInfo.Filename,
				CreatedAt: time.Now().UnixMilli(),
				Sha256:    fileInfo.Sha256,
			})
			inputMetadata.SetFiles(files)
			inputMetadata.UpdateSize(fileInfo.Size)
		}
	} else {
		files = append(files, metadata.File{
			Size: fileInfo.Size, Filename: fileInfo.Filename,
			Sha256: fileInfo.Sha256, CreatedAt: time.Now().UnixMilli(),
		})
		inputMetadata.SetFiles(files)
		inputMetadata.UpdateSize(fileInfo.Size)
	}
	return nil
}

func (l *localBase) CheckIfFileAlreadyExist(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version string,
	metadata metadata.Metadata,
	fileName string,
	path string,
) error {
	image, err := l.imageDao.GetByName(ctx, info.RegistryID, info.Image)
	if err != nil && !strings.Contains(err.Error(), "resource not found") {
		return fmt.Errorf("failed to fetch the image for artifact : [%s] with registry : [%s]", info.Image,
			info.RegIdentifier)
	}
	if image == nil {
		return nil
	}

	dbArtifact, err := l.artifactDao.GetByName(ctx, image.ID, version)

	if err != nil && !strings.Contains(err.Error(), "resource not found") {
		return fmt.Errorf("failed to fetch artifact : [%s] with registry : [%s]", info.Image, info.RegIdentifier)
	}

	if dbArtifact == nil {
		return nil
	}

	err = json.Unmarshal(dbArtifact.Metadata, metadata)
	if err != nil {
		return fmt.Errorf("failed to get metadata for artifact: [%s] with registry: [%s] and error: %w", info.Image,
			info.RegIdentifier, err)
	}

	for _, file := range metadata.GetFiles() {
		if file.Filename == fileName && l.Exists(ctx, info, path) {
			return errors.Conflictf("file: [%s] for Artifact: [%s], Version: [%s] and registry: [%s] already exist",
				fileName, info.Image, version, info.RegIdentifier)
		}
	}

	return nil
}

func (l *localBase) DeletePackage(ctx context.Context, info pkg.PackageArtifactInfo) error {
	err := l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			path := "/" + info.BaseArtifactInfo().Image
			err := l.nodesDao.DeleteByNodePathAndRegistryID(ctx, path, info.BaseArtifactInfo().RegistryID)

			if err != nil {
				return err
			}

			err = l.artifactDao.DeleteByImageNameAndRegistryID(ctx,
				info.BaseArtifactInfo().RegistryID, info.BaseArtifactInfo().Image)

			if err != nil {
				return err
			}

			err = l.imageDao.DeleteByImageNameAndRegID(ctx, info.BaseArtifactInfo().RegistryID,
				info.BaseArtifactInfo().Image)

			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to delete the package %v", info.BaseArtifactInfo().Image)
		return err
	}

	return nil
}

func (l *localBase) DeleteVersion(ctx context.Context, info pkg.PackageArtifactInfo) error {
	err := l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			path := "/" + info.BaseArtifactInfo().Image + "/" + info.GetVersion()
			err := l.nodesDao.DeleteByNodePathAndRegistryID(ctx,
				path, info.BaseArtifactInfo().RegistryID)

			if err != nil {
				return err
			}

			err = l.artifactDao.DeleteByVersionAndImageName(ctx,
				info.BaseArtifactInfo().Image, info.GetVersion(), info.BaseArtifactInfo().RegistryID)
			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to delete the version for artifact %v:%v", info.BaseArtifactInfo().Image,
			info.GetVersion())
		return err
	}
	return nil
}

// AuditPush is a convenience wrapper for calling the centralized audit function from localBase.
func (l *localBase) AuditPush(
	ctx context.Context, info pkg.ArtifactInfo, version string,
	imageUUID string, artifactUUID string,
) {
	registryaudit.LogArtifactPush(
		ctx, l.auditService, l.spaceFinder, info, version, imageUUID, artifactUUID,
	)
}
