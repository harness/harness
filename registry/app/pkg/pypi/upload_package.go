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

package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store/database"
	"github.com/harness/gitness/registry/types"
)

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (c *controller) UploadPackageFile(
	ctx context.Context,
	info ArtifactInfo,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (*commons.ResponseHeaders, string, errcode.Error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	// Fixme: Generalize this check for all package types
	// err = c.CheckIfFileAlreadyExist(ctx, info)
	//
	// if err != nil {
	//	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(err)
	// }

	registry, err := c.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	fileName := fileHeader.Filename
	path := info.Image + "/" + info.Metadata.Version + "/" + fileName
	fileInfo, err := c.fileManager.UploadFile(ctx, path, info.RegIdentifier, registry.ID,
		info.RootParentID, info.RootIdentifier, file, nil, fileName)
	if err != nil {
		return responseHeaders, "", errcode.ErrCodeUnknown.WithDetail(err)
	}
	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			image := &types.Image{
				Name:       info.Image,
				RegistryID: registry.ID,
				Enabled:    true,
			}
			err := c.imageDao.CreateOrUpdate(ctx, image)
			if err != nil {
				return fmt.Errorf("failed to create image for artifact: [%s], error: %w", info.Image, err)
			}

			dbArtifact, err := c.artifactDao.GetByName(ctx, image.ID, info.Metadata.Version)

			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				return fmt.Errorf("failed to fetch artifact : [%s] with error: %w", info.Image, err)
			}

			metadata := &database.PyPiMetadata{
				Metadata: info.Metadata,
			}
			err2 := c.updateMetadata(dbArtifact, metadata, info, fileInfo)
			if err2 != nil {
				return fmt.Errorf("failed to update metadata for artifact: [%s] with error: %w", info.Image, err2)
			}

			metadataJSON, err := json.Marshal(metadata)

			if err != nil {
				return fmt.Errorf("failed to parse metadata for artifact: [%s] with error: %w", info.Image, err)
			}

			err = c.artifactDao.CreateOrUpdate(ctx, &types.Artifact{
				ImageID:  image.ID,
				Version:  info.Metadata.Version,
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

func (c *controller) updateMetadata(
	dbArtifact *types.Artifact, metadata *database.PyPiMetadata,
	info ArtifactInfo, fileInfo pkg.FileInfo,
) error {
	var files []database.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, metadata)
		if err != nil {
			return fmt.Errorf("failed to get metadata for artifact: [%s] with registry: [%s] and error: %w", info.Image,
				info.RegIdentifier, err)
		}
		fileExist := false
		files = metadata.Files
		for _, file := range files {
			if file.Filename == fileInfo.Filename {
				fileExist = true
			}
		}
		if !fileExist {
			files = append(files, database.File{
				Size: fileInfo.Size, Filename: fileInfo.Filename,
				CreatedAt: time.Now().UnixMilli(),
			})
			metadata.Files = files
		}
	} else {
		files = append(files, database.File{
			Size: fileInfo.Size, Filename: fileInfo.Filename,
			CreatedAt: time.Now().UnixMilli(),
		})
		metadata.Files = files
	}
	return nil
}
