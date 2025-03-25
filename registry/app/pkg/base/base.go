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
	"strings"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
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
	) (
		// TODO: Check the scope if we should remove the response message / headers setup here or
		// each package implementation should have their own.
		headers *commons.ResponseHeaders, sha256 string, err errcode.Error,
	)
	Upload(
		ctx context.Context,
		info pkg.ArtifactInfo,
		fileName string,
		version string,
		path string,
		file io.ReadCloser,
		metadata metadata.Metadata,
	) (*commons.ResponseHeaders, string, errcode.Error)
	Download(ctx context.Context, info pkg.ArtifactInfo, version string, fileName string) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		[]error,
	)

	Exists(ctx context.Context, info pkg.ArtifactInfo, version string, fileName string) bool
}

type localBase struct {
	registryDao store.RegistryRepository
	fileManager filemanager.FileManager
	tx          dbtx.Transactor
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
}

func NewLocalBase(
	registryDao store.RegistryRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
) LocalBase {
	return &localBase{
		registryDao: registryDao,
		fileManager: fileManager,
		tx:          tx,
		imageDao:    imageDao,
		artifactDao: artifactDao,
	}
}

func (l *localBase) UploadFile(
	ctx context.Context,
	info pkg.ArtifactInfo,
	fileName string,
	version string,
	path string,
	file multipart.File,
	// TODO: Metadata shouldn't be provided as a parameter, it should be fetched or created.
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, errcode.Error) {
	return l.uploadInternal(ctx, info, fileName, version, path, file, nil, metadata)
}

func (l *localBase) Upload(
	ctx context.Context,
	info pkg.ArtifactInfo,
	fileName string,
	version string,
	path string,
	file io.ReadCloser,
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, errcode.Error) {
	return l.uploadInternal(ctx, info, fileName, version, path, nil, file, metadata)
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
) (*commons.ResponseHeaders, string, errcode.Error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	err := l.CheckIfFileAlreadyExist(ctx, info, version, metadata, fileName)

	if err != nil {
		return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	registry, err := l.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	fileInfo, err := l.fileManager.UploadFile(ctx, path, info.RegIdentifier, registry.ID,
		info.RootParentID, info.RootIdentifier, file, fileReadCloser, fileName)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	err = l.tx.WithTx(
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

			dbArtifact, err := l.artifactDao.GetByName(ctx, image.ID, version)

			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				return fmt.Errorf("failed to fetch artifact : [%s] with error: %w", info.Image, err)
			}

			err2 := l.updateMetadata(dbArtifact, metadata, info, fileInfo)
			if err2 != nil {
				return fmt.Errorf("failed to update metadata for artifact: [%s] with error: %w", info.Image, err2)
			}

			metadataJSON, err := json.Marshal(metadata)

			if err != nil {
				return fmt.Errorf("failed to parse metadata for artifact: [%s] with error: %w", info.Image, err)
			}

			err = l.artifactDao.CreateOrUpdate(ctx, &types.Artifact{
				ImageID:  image.ID,
				Version:  version,
				Metadata: metadataJSON,
			})
			if err != nil {
				return fmt.Errorf("failed to create artifact : [%s] with error: %w", info.Image, err)
			}
			return nil
		})

	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	responseHeaders.Code = http.StatusCreated
	return responseHeaders, fileInfo.Sha256, errcode.Error{}
}

func (l *localBase) Download(ctx context.Context, info pkg.ArtifactInfo, version string, fileName string) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	[]error,
) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	path := "/" + info.Image + "/" + version + "/" + fileName
	reg, _ := l.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)

	fileReader, _, redirectURL, err := l.fileManager.DownloadFile(ctx, path, types.Registry{
		ID:   reg.ID,
		Name: info.RegIdentifier,
	}, info.RootIdentifier)
	if err != nil {
		return responseHeaders, nil, "", []error{err}
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, nil
}

func (l *localBase) Exists(ctx context.Context, info pkg.ArtifactInfo, version string, fileName string) bool {
	filePath := "/" + info.Image + "/" + version + "/" + fileName
	sha256, _ := l.fileManager.HeadFile(ctx, filePath, info.RegistryID)
	//FIXME: err should be checked on if the record doesn't exist or there was DB call issue
	return sha256 != ""
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
			})
			inputMetadata.SetFiles(files)
		}
	} else {
		files = append(files, metadata.File{
			Size: fileInfo.Size, Filename: fileInfo.Filename,
			CreatedAt: time.Now().UnixMilli(),
		})
		inputMetadata.SetFiles(files)
	}
	return nil
}

func (l *localBase) CheckIfFileAlreadyExist(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version string,
	metadata metadata.Metadata,
	fileName string,
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
		if file.Filename == fileName {
			return fmt.Errorf("file: [%s] with Artifact: [%s], Version: [%s] and registry: [%s] already exist",
				fileName, info.Image, version, info.RegIdentifier)
		}
	}

	return nil
}
