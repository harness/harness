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
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"
)

const (
	duplicateFileError = "file: [%s] with Artifact: [%s], Version: [%s] and registry: [%s] already exist"
	filenameRegex      = `^[a-zA-Z0-9][a-zA-Z0-9._~@,/-]*[a-zA-Z0-9]$`
)

// ErrDuplicateFile is returned when a file already exists in the registry.
var ErrDuplicateFile = errors.New("duplicate file error")

type Controller struct {
	SpaceStore  corestore.SpaceStore
	Authorizer  authz.Authorizer
	DBStore     *DBStore
	fileManager filemanager.FileManager
	tx          dbtx.Transactor
	spaceFinder refcache.SpaceFinder
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
	spaceFinder refcache.SpaceFinder,
) *Controller {
	return &Controller{
		SpaceStore:  spaceStore,
		Authorizer:  authorizer,
		fileManager: fileManager,
		DBStore:     dBStore,
		tx:          tx,
		spaceFinder: spaceFinder,
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

func (c Controller) UploadArtifact(
	ctx context.Context, info pkg.GenericArtifactInfo,
	r *http.Request,
) (*commons.ResponseHeaders, string, errcode.Error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	err := pkg.GetRegistryCheckAccess(ctx, c.Authorizer, c.spaceFinder, info.ParentID, *info.ArtifactInfo,
		enum.PermissionArtifactsUpload)
	if err != nil {
		return nil, "", errcode.ErrCodeDenied.WithDetail(err)
	}

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", errcode.ErrCodeUnknown.WithDetail(err)
	}

	fileInfo, err := c.UploadFile(ctx, reader, &info)

	if errors.Is(err, ErrDuplicateFile) {
		return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

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

			metadata := &metadata.GenericMetadata{
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

			_, err = c.DBStore.ArtifactDao.CreateOrUpdate(ctx, &types.Artifact{
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

func (c Controller) updateMetadata(
	dbArtifact *types.Artifact, metadataInput *metadata.GenericMetadata,
	info pkg.GenericArtifactInfo, fileInfo types.FileInfo,
) error {
	var files []metadata.File
	if dbArtifact != nil {
		err := json.Unmarshal(dbArtifact.Metadata, metadataInput)
		if err != nil {
			return fmt.Errorf("failed to get metadata for artifact : [%s] with registry : [%s]", info.Image,
				info.RegIdentifier)
		}
		fileExist := false
		files = metadataInput.Files
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
			metadataInput.Files = files
			metadataInput.FileCount++
			metadataInput.Size += fileInfo.Size
		}
	} else {
		files = append(files, metadata.File{
			Size: fileInfo.Size, Filename: fileInfo.Filename,
			CreatedAt: time.Now().UnixMilli(),
		})
		metadataInput.Files = files
		metadataInput.FileCount++
		metadataInput.Size += fileInfo.Size
	}
	return nil
}

func (c Controller) PullArtifact(ctx context.Context, info pkg.GenericArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader, string, errcode.Error,
) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	err := pkg.GetRegistryCheckAccess(ctx, c.Authorizer, c.spaceFinder, info.ParentID, *info.ArtifactInfo,
		enum.PermissionArtifactsDownload)
	if err != nil {
		return nil, nil, "", errcode.ErrCodeDenied.WithDetail(err)
	}

	path := "/" + info.Image + "/" + info.Version + "/" + info.FileName
	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(ctx, path, info.RegistryID,
		info.RegIdentifier, info.RootIdentifier, true)
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

	metadata := &metadata.GenericMetadata{}

	err = json.Unmarshal(dbArtifact.Metadata, metadata)

	if err == nil {
		for _, file := range metadata.Files {
			if file.Filename == info.FileName {
				return fmt.Errorf("%w: %s", ErrDuplicateFile,
					fmt.Sprintf(duplicateFileError, info.FileName, info.Image, info.Version, info.RegIdentifier))
			}
		}
	}
	return nil
}

func (c Controller) ParseAndUploadToTmp(
	ctx context.Context, reader *multipart.Reader,
	info pkg.GenericArtifactInfo, fileToFind string,
	formKeys []string,
) (types.FileInfo, string, map[string]string, error) {
	formValues := make(map[string]string)
	var fileInfo types.FileInfo
	var filename string

	// Track which keys we still need to find
	keysToFind := make(map[string]bool)
	for _, key := range formKeys {
		keysToFind[key] = true
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return types.FileInfo{}, "", formValues, err
		}

		formName := part.FormName()

		// Check if this is the file part we're looking for
		if formName == fileToFind {
			fileInfo, filename, err = c.fileManager.UploadTempFile(ctx, info.RootIdentifier, nil, "", part)

			if err != nil {
				return types.FileInfo{}, "", formValues, err
			}
			continue
		}

		// Check if this is one of the form values we're looking for
		if _, ok := keysToFind[formName]; !ok {
			// Not a key we're looking for
			part.Close()
			continue
		}

		// Read the form value (these are typically small)
		var valueBytes []byte
		buffer := make([]byte, 1024)

		for {
			n, err := part.Read(buffer)
			if n > 0 {
				valueBytes = append(valueBytes, buffer[:n]...)
			}
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return types.FileInfo{}, "", formValues, fmt.Errorf("error reading form value: %w", err)
			}
		}

		formValues[formName] = string(valueBytes)
		delete(keysToFind, formName)
		part.Close()

		// If we've found all form keys and the file (if needed), we can stop
		if len(keysToFind) == 0 {
			break
		}
	}

	// If fileToFind was provided but not found, return an error
	if fileToFind != "" && filename == "" {
		return types.FileInfo{}, "", formValues, fmt.Errorf("file part with key '%s' not found", fileToFind)
	}

	return fileInfo, filename, formValues, nil
}

func (c Controller) UploadFile(
	ctx context.Context, reader *multipart.Reader,
	info *pkg.GenericArtifactInfo,
) (types.FileInfo, error) {
	fileInfo, tmpFileName, formValues, err :=
		c.ParseAndUploadToTmp(ctx, reader, *info, "file", []string{"filename", "description"})

	if err != nil {
		return types.FileInfo{},
			fmt.Errorf("failed to Parse/upload "+
				"the generic artifact file to temp location: [%s] with error: [%w] ", tmpFileName, err)
	}

	if err := validateFileName(formValues["filename"]); err != nil {
		return types.FileInfo{},
			fmt.Errorf("invalid file name for generic artifact file: [%s]", formValues["filename"])
	}

	info.FileName = formValues["filename"]
	info.Description = formValues["description"]
	fileInfo.Filename = info.FileName

	err = c.CheckIfFileAlreadyExist(ctx, *info)

	if err != nil {
		return types.FileInfo{},
			fmt.Errorf("file already exist: [%w] ", err)
	}

	session, _ := request.AuthSessionFrom(ctx)
	filePath := path.Join(info.Image, info.Version, fileInfo.Filename)
	err = c.fileManager.MoveTempFile(ctx, filePath, info.RegistryID,
		info.RootParentID, info.RootIdentifier, fileInfo, tmpFileName, session.Principal.ID)
	if err != nil {
		return types.FileInfo{}, err
	}
	return fileInfo, err
}

func validateFileName(filename string) error {
	filenameRe := regexp.MustCompile(filenameRegex)

	if !filenameRe.MatchString(filename) {
		return fmt.Errorf("invalid filename: %s", filename)
	}
	return nil
}
