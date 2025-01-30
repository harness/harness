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

package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	spaceStore  corestore.SpaceStore
	authorizer  authz.Authorizer
	DBStore     *DBStore
	fileManager filemanager.FileManager
	tx          dbtx.Transactor
}

type DBStore struct {
	RegistryDao      store.RegistryRepository
	ImageDao         store.ImageRepository
	ArtifactDao      store.ArtifactRepository
	TagDao           store.TagRepository
	BandwidthStatDao store.BandwidthStatRepository
	DownloadStatDao  store.DownloadStatRepository
}

func NewController(
	spaceStore corestore.SpaceStore,
	authorizer authz.Authorizer,
	fileManager filemanager.FileManager,
	dBStore *DBStore,
	tx dbtx.Transactor,
) *Controller {
	return &Controller{
		spaceStore:  spaceStore,
		authorizer:  authorizer,
		fileManager: fileManager,
		DBStore:     dBStore,
		tx:          tx,
	}
}

func NewDBStore(
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
) *DBStore {
	return &DBStore{
		RegistryDao:      registryDao,
		ImageDao:         imageDao,
		ArtifactDao:      artifactDao,
		BandwidthStatDao: bandwidthStatDao,
		DownloadStatDao:  downloadStatDao,
	}
}

const regNameFormat = "registry : [%s]"

func (c Controller) UploadArtifact(ctx context.Context, info pkg.GenericArtifactInfo,
	file multipart.File) (*commons.ResponseHeaders, string, errcode.Error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	err := pkg.GetRegistryCheckAccess(
		ctx, c.DBStore.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsUpload,
	)
	if err != nil {
		return nil, "", errcode.ErrCodeDenied.WithDetail(err)
	}

	err = c.CheckIfFileAlreadyExist(ctx, info)

	if err != nil {
		return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	path := info.Image + "/" + info.Version + "/" + info.FileName
	fileInfo, err := c.fileManager.UploadFile(ctx, path, info.RegIdentifier, info.RegistryID,
		info.RootParentID, info.RootIdentifier, file, nil, info.FileName)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			image := &types.Image{
				Name:       info.Image,
				RegistryID: info.RegistryID,
				Enabled:    true,
			}
			err := c.DBStore.ImageDao.CreateOrUpdate(ctx, image)
			if err != nil {
				return fmt.Errorf("failed to create image for artifact : [%s] with "+
					regNameFormat, info.Image, info.RegIdentifier)
			}

			dbArtifact, err := c.DBStore.ArtifactDao.GetByName(ctx, image.ID, info.Version)

			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				return fmt.Errorf("failed to fetch artifact : [%s] with "+
					regNameFormat, info.Image, info.RegIdentifier)
			}

			metadata := &database.GenericMetadata{
				Description: info.Description,
			}
			err2 := c.updateMetadata(dbArtifact, metadata, info, fileInfo)
			if err2 != nil {
				return fmt.Errorf("failed to update metadata for artifact : [%s] with "+
					regNameFormat, info.Image, info.RegIdentifier)
			}

			metadataJSON, err := json.Marshal(metadata)

			if err != nil {
				return fmt.Errorf("failed to parse metadata for artifact : [%s] with "+
					regNameFormat, info.Image, info.RegIdentifier)
			}

			err = c.DBStore.ArtifactDao.CreateOrUpdate(ctx, &types.Artifact{
				ImageID:  image.ID,
				Version:  info.Version,
				Metadata: metadataJSON,
			})
			if err != nil {
				return fmt.Errorf("failed to create artifact : [%s] with "+
					regNameFormat, info.Image, info.RegIdentifier)
			}
			return nil
		})

	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	responseHeaders.Code = http.StatusCreated
	return responseHeaders, fileInfo.Sha256, errcode.Error{}
}

func (c Controller) updateMetadata(dbArtifact *types.Artifact, metadata *database.GenericMetadata,
	info pkg.GenericArtifactInfo, fileInfo pkg.FileInfo) error {
	var files []database.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, metadata)
		if err != nil {
			return fmt.Errorf("failed to get metadata for artifact : [%s] with registry : [%s]", info.Image, info.RegIdentifier)
		}
		fileExist := false
		files = metadata.Files
		for _, file := range files {
			if file.Filename == info.FileName {
				fileExist = true
			}
		}
		if !fileExist {
			files = append(files, database.File{Size: fileInfo.Size, Filename: fileInfo.Filename,
				CreatedAt: time.Now().UnixMilli()})
			metadata.Files = files
			metadata.FileCount++
		}
	} else {
		files = append(files, database.File{Size: fileInfo.Size, Filename: fileInfo.Filename,
			CreatedAt: time.Now().UnixMilli()})
		metadata.Files = files
		metadata.FileCount++
	}
	return nil
}

func (c Controller) PullArtifact(ctx context.Context, info pkg.GenericArtifactInfo) (*commons.ResponseHeaders,
	*storage.FileReader, string, errcode.Error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	err := pkg.GetRegistryCheckAccess(
		ctx, c.DBStore.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, nil, "", errcode.ErrCodeDenied.WithDetail(err)
	}

	path := "/" + info.Image + "/" + info.Version + "/" + info.FileName
	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(ctx, path, types.Registry{
		ID:   info.RegistryID,
		Name: info.RegIdentifier,
	}, info.RootIdentifier)
	if err != nil {
		return responseHeaders, nil, "", errcode.ErrCodeRootNotFound.WithDetail(err)
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, errcode.Error{}
}

func (c Controller) CheckIfFileAlreadyExist(ctx context.Context, info pkg.GenericArtifactInfo) error {
	image, err := c.DBStore.ImageDao.GetByName(ctx, info.RegistryID, info.Image)
	if err != nil && !strings.Contains(err.Error(), "resource not found") {
		return fmt.Errorf("failed to fetch the image for artifact : [%s] with "+
			regNameFormat, info.Image, info.RegIdentifier)
	}
	if image == nil {
		return nil
	}

	dbArtifact, err := c.DBStore.ArtifactDao.GetByName(ctx, image.ID, info.Version)

	if err != nil && !strings.Contains(err.Error(), "resource not found") {
		return fmt.Errorf("failed to fetch artifact : [%s] with "+
			regNameFormat, info.Image, info.RegIdentifier)
	}

	if dbArtifact == nil {
		return nil
	}

	metadata := &database.GenericMetadata{}

	err = json.Unmarshal(dbArtifact.Metadata, metadata)

	if err == nil {
		for _, file := range metadata.Files {
			if file.Filename == info.FileName {
				return fmt.Errorf("file: [%s] with Artifact: [%s], Version: [%s] and registry: [%s] already exist",
					info.FileName, info.Image, info.Version, info.RegIdentifier)
			}
		}
	}
	return nil
}
