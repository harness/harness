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

package maven

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/maven/utils"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
)

const (
	ArtifactTypeLocalRegistry = "Local Registry"
)

func NewLocalRegistry(
	dBStore *DBStore, tx dbtx.Transactor,

	fileManager filemanager.FileManager,
) Registry {
	return &LocalRegistry{
		DBStore:     dBStore,
		tx:          tx,
		fileManager: fileManager,
	}
}

type LocalRegistry struct {
	DBStore     *DBStore
	tx          dbtx.Transactor
	fileManager filemanager.FileManager
}

func (r *LocalRegistry) GetMavenArtifactType() string {
	return ArtifactTypeLocalRegistry
}

func (r *LocalRegistry) HeadArtifact(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, errs []error,
) {
	responseHeaders, _, _, _, errs = r.FetchArtifact(ctx, info, false)
	return responseHeaders, errs
}

func (r *LocalRegistry) GetArtifact(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	return r.FetchArtifact(ctx, info, true)
}

func (r *LocalRegistry) FetchArtifact(ctx context.Context, info pkg.MavenArtifactInfo, serveFile bool) (
	responseHeaders *commons.ResponseHeaders,
	body *storage.FileReader,
	readCloser io.ReadCloser,
	redirectURL string,
	errs []error,
) {
	filePath := utils.GetFilePath(info)
	name := info.GroupID + ":" + info.ArtifactID
	dbImage, err2 := r.DBStore.ImageDao.GetByName(ctx, info.RegistryID, name)
	if err2 != nil {
		return processError(err2)
	}
	if utils.IsSnapshotVersion(info) {
		node, err := r.DBStore.NodeDao.FindByPathAndRegistryID(ctx, info.RegistryID, utils.AddLikeBeforeExtension(info))
		if err != nil {
			return processError(err)
		}
		info.FileName = node.Name
		filePath = utils.GetFilePath(info)
	}
	_, err2 = r.DBStore.ArtifactDao.GetByName(ctx, dbImage.ID, info.Version)
	if err2 != nil {
		return processError(err2)
	}

	fileInfo, err := r.fileManager.GetFileMetadata(ctx, filePath, info.RegistryID)
	if err != nil {
		return processError(err)
	}
	var fileReader *storage.FileReader
	if serveFile {
		fileReader, _, redirectURL, err = r.fileManager.DownloadFile(ctx, filePath, types.Registry{
			ID:   info.RegistryID,
			Name: info.RootIdentifier,
		}, info.RootIdentifier)
		if err != nil {
			return processError(err)
		}
	}
	responseHeaders = utils.SetHeaders(info, fileInfo)
	return responseHeaders, fileReader, nil, redirectURL, nil
}

func (r *LocalRegistry) PutArtifact(ctx context.Context, info pkg.MavenArtifactInfo, fileReader io.Reader) (
	responseHeaders *commons.ResponseHeaders, errs []error,
) {
	filePath := utils.GetFilePath(info)
	fileInfo, err := r.fileManager.UploadFile(ctx, filePath, info.RegIdentifier,
		info.RegistryID, info.RootParentID, info.RootIdentifier, nil, fileReader, info.FileName)
	if err != nil {
		return responseHeaders, []error{errcode.ErrCodeUnknown.WithDetail(err)}
	}
	err = r.tx.WithTx(
		ctx, func(ctx context.Context) error {
			name := info.GroupID + ":" + info.ArtifactID
			dbImage := &types.Image{
				Name:       name,
				RegistryID: info.RegistryID,
				Enabled:    true,
			}

			err2 := r.DBStore.ImageDao.CreateOrUpdate(ctx, dbImage)
			if err2 != nil {
				return err2
			}

			if info.Version == "" {
				return nil
			}

			metadata := &metadata.MavenMetadata{}

			dbArtifact, err3 := r.DBStore.ArtifactDao.GetByName(ctx, dbImage.ID, info.Version)

			if err3 != nil && !strings.Contains(err3.Error(), "resource not found") {
				return err3
			}

			err3 = r.updateArtifactMetadata(dbArtifact, metadata, info, fileInfo)
			if err3 != nil {
				return err3
			}

			metadataJSON, err3 := json.Marshal(metadata)

			if err3 != nil {
				return err3
			}

			dbArtifact = &types.Artifact{
				ImageID:  dbImage.ID,
				Version:  info.Version,
				Metadata: metadataJSON,
			}

			err2 = r.DBStore.ArtifactDao.CreateOrUpdate(ctx, dbArtifact)
			if err2 != nil {
				return err2
			}

			return nil
		})

	if err != nil {
		return responseHeaders, []error{errcode.ErrCodeUnknown.WithDetail(err)}
	}
	responseHeaders = &commons.ResponseHeaders{
		Headers: map[string]string{},
		Code:    http.StatusCreated,
	}
	return responseHeaders, nil
}

func (r *LocalRegistry) updateArtifactMetadata(
	dbArtifact *types.Artifact, mavenMetadata *metadata.MavenMetadata,
	info pkg.MavenArtifactInfo, fileInfo pkg.FileInfo,
) error {
	var files []metadata.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, mavenMetadata)
		if err != nil {
			return err
		}
		fileExist := false
		files = mavenMetadata.Files
		for _, file := range files {
			if file.Filename == info.FileName {
				fileExist = true
			}
		}
		if !fileExist {
			files = append(files, metadata.File{
				Size: fileInfo.Size, Filename: fileInfo.Filename,
				CreatedAt: time.Now().UnixMilli(),
			})
			mavenMetadata.Files = files
			mavenMetadata.FileCount++
		}
	} else {
		files = append(files, metadata.File{
			Size: fileInfo.Size, Filename: fileInfo.Filename,
			CreatedAt: time.Now().UnixMilli(),
		})
		mavenMetadata.Files = files
		mavenMetadata.FileCount++
	}
	return nil
}

func processError(err error) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	if strings.Contains(err.Error(), sql.ErrNoRows.Error()) ||
		strings.Contains(err.Error(), "resource not found") ||
		strings.Contains(err.Error(), "http status code: 404") {
		return responseHeaders, nil, nil, "", []error{commons.NotFoundError(err.Error(), err)}
	}
	return responseHeaders, nil, nil, "", []error{errcode.ErrCodeUnknown.WithDetail(err)}
}
