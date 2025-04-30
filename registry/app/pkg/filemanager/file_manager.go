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

package filemanager

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"

	"github.com/harness/gitness/registry/app/event"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	rootPathString = "/"
	tmp            = "tmp"
	files          = "files"
	nodeLimit      = 1000
	pathFormat     = "for path: %s, with error %w"

	failedToGetFile = "failed to get the file for path: %s, with error %w"
)

func NewFileManager(
	app *App, registryDao store.RegistryRepository, genericBlobDao store.GenericBlobRepository,
	nodesDao store.NodesRepository,
	tx dbtx.Transactor, reporter event.Reporter,

) FileManager {
	return FileManager{
		App:            app,
		registryDao:    registryDao,
		genericBlobDao: genericBlobDao,
		nodesDao:       nodesDao,
		tx:             tx,
		reporter:       reporter,
	}
}

type FileManager struct {
	App            *App
	registryDao    store.RegistryRepository
	genericBlobDao store.GenericBlobRepository
	nodesDao       store.NodesRepository
	tx             dbtx.Transactor
	reporter       event.Reporter
}

func (f *FileManager) UploadFile(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	rootIdentifier string,
	file multipart.File,
	fileReader io.Reader,
	fileName string,
) (types.FileInfo, error) {
	// uploading the file to temporary path in file storage
	blobContext := f.App.GetBlobsContext(ctx, rootIdentifier)
	tmpFileName := uuid.NewString()
	fileInfo, tmpPath, err := f.uploadTempFileInternal(ctx, blobContext, rootIdentifier,
		file, fileName, fileReader, tmpFileName)
	if err != nil {
		return fileInfo, err
	}
	fileInfo.Filename = fileName

	err = f.moveFile(ctx, rootIdentifier, fileInfo, blobContext, tmpPath)
	if err != nil {
		return fileInfo, err
	}

	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo)
	if err != nil {
		return fileInfo, err
	}

	// Emit blob create event
	if created {
		event.ReportEventAsync(ctx, rootIdentifier, f.reporter, event.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.App.Config)
	}
	return fileInfo, nil
}

func (f *FileManager) dbSaveFile(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	fileInfo types.FileInfo,
) (string, bool, error) {
	// Saving in the generic blobs table
	var blobID = ""
	gb := &types.GenericBlob{
		RootParentID: rootParentID,
		Sha1:         fileInfo.Sha1,
		Sha256:       fileInfo.Sha256,
		Sha512:       fileInfo.Sha512,
		MD5:          fileInfo.MD5,
		Size:         fileInfo.Size,
	}
	created, err := f.genericBlobDao.Create(ctx, gb)
	if err != nil {
		log.Error().Msgf("failed to save generic blob in db with "+
			"sha256 : %s, err: %s", fileInfo.Sha256, err.Error())
		return "", false, fmt.Errorf("failed to save generic blob"+
			" in db with sha256 : %s, err: %w", fileInfo.Sha256, err)
	}
	blobID = gb.ID
	// Saving the nodes
	err = f.tx.WithTx(ctx, func(ctx context.Context) error {
		err = f.createNodes(ctx, filePath, blobID, regID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Msgf("failed to save nodes for file : %s, with "+
			"path : %s, err: %s", fileInfo.Filename, filePath, err)
		return "", false, fmt.Errorf("failed to save nodes for"+
			" file : %s, with path : %s, err: %w", fileInfo.Filename, filePath, err)
	}
	return blobID, created, nil
}

func (f *FileManager) moveFile(
	ctx context.Context,
	rootIdentifier string,
	fileInfo types.FileInfo,
	blobContext *Context,
	tmpPath string,
) error {
	// Moving the file to permanent path in file storage
	fileStoragePath := path.Join(rootPathString, rootIdentifier, files, fileInfo.Sha256)
	err := blobContext.genericBlobStore.Move(ctx, tmpPath, fileStoragePath)

	if err != nil {
		log.Error().Msgf("failed to Move the file on permanent location "+
			"with name : %s with error : %s", fileInfo.Filename, err.Error())
		return fmt.Errorf("failed to Move the file on permanent"+
			" location with name : %s with error : %w", fileInfo.Filename, err)
	}
	return nil
}

func (f *FileManager) createNodes(ctx context.Context, filePath string, blobID string, regID int64) error {
	segments := strings.Split(filePath, rootPathString)
	parentID := ""
	// Start with root (-1)
	// Iterate through segments and create Node objects
	nodePath := ""
	for i, segment := range segments {
		if i >= nodeLimit { // Stop after 1000 iterations
			break
		}
		if segment == "" {
			continue // Skip empty segments
		}
		var nodeID string
		var err error
		nodePath += rootPathString + segment
		if i == len(segments)-1 {
			nodeID, err = f.SaveNode(ctx, filePath, blobID, regID, segment,
				parentID, nodePath, true)
			if err != nil {
				return err
			}
		} else {
			nodeID, err = f.SaveNode(ctx, filePath, "", regID, segment,
				parentID, nodePath, false)
			if err != nil {
				return err
			}
		}
		parentID = nodeID
	}
	return nil
}

func (f *FileManager) SaveNode(
	ctx context.Context, filePath string, blobID string, regID int64, segment string,
	parentID string, nodePath string, isFile bool,
) (string, error) {
	node := &types.Node{
		Name:         segment,
		RegistryID:   regID,
		ParentNodeID: parentID,
		IsFile:       isFile,
		NodePath:     nodePath,
		BlobID:       blobID,
	}
	err := f.nodesDao.Create(ctx, node)
	if err != nil {
		return "", fmt.Errorf("failed to create the node: %s, "+
			"for path := %s, %w", segment, filePath, err)
	}
	return node.ID, nil
}

func (f *FileManager) DownloadFile(
	ctx context.Context,
	filePath string,
	registryID int64,
	registryIdentifier string,
	rootIdentifier string,
) (fileReader *storage.FileReader, size int64, redirectURL string, err error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, registryID, filePath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get the file for path: %s, "+
			"with registry: %s", filePath, registryIdentifier)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get the blob for path: %s, "+
			"with blob id: %s, with error %s", filePath, node.BlobID, err)
	}

	completeFilaPath := path.Join(rootPathString + rootIdentifier + rootPathString + files + rootPathString + blob.Sha256)
	blobContext := f.App.GetBlobsContext(ctx, rootIdentifier)
	reader, redirectURL, err := blobContext.genericBlobStore.Get(ctx, completeFilaPath, blob.Size)

	if err != nil {
		return nil, 0, "", fmt.Errorf(failedToGetFile, completeFilaPath, err)
	}

	if redirectURL != "" {
		return reader, blob.Size, redirectURL, nil
	}

	return reader, blob.Size, "", nil
}

func (f *FileManager) DeleteNode(
	ctx context.Context,
	regID int64,
	filePath string,
) error {
	err := f.nodesDao.DeleteByNodePathAndRegistryID(ctx, filePath, regID)
	if err != nil {
		return fmt.Errorf("failed to delete file for path: %s, with error: %w", filePath, err)
	}
	return nil
}

func (f *FileManager) HeadFile(
	ctx context.Context,
	filePath string,
	regID int64,
) (string, error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, regID, filePath)

	if err != nil {
		return "", fmt.Errorf("failed to get the node path mapping for path: %s, "+
			"with error %w", filePath, err)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return "", fmt.Errorf("failed to get the blob for path: %s, with blob id: %s,"+
			" with error %w", filePath, node.BlobID, err)
	}
	return blob.Sha256, nil
}

func (f *FileManager) GetFileMetadata(
	ctx context.Context,
	filePath string,
	regID int64,
) (types.FileInfo, error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, regID, filePath)

	if err != nil {
		return types.FileInfo{}, fmt.Errorf("failed to get the node path mapping "+
			pathFormat, filePath, err)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return types.FileInfo{}, fmt.Errorf("failed to get the blob for path: %s, "+
			"with blob id: %s, with error %s", filePath, node.BlobID, err)
	}
	return types.FileInfo{
		Sha1:     blob.Sha1,
		Size:     blob.Size,
		Sha256:   blob.Sha256,
		Sha512:   blob.Sha512,
		MD5:      blob.MD5,
		Filename: node.Name,
	}, nil
}

func (f *FileManager) DeleteFileByRegistryID(
	ctx context.Context,
	regID int64,
	regName string,
) error {
	err := f.nodesDao.DeleteByRegistryID(ctx, regID)
	if err != nil {
		return fmt.Errorf("failed to delete all the files for registry with name: %s, with error %w", regName, err)
	}
	return nil
}

func (f *FileManager) GetFilesMetadata(
	ctx context.Context,
	filePath string,
	regID int64,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
) (*[]types.FileNodeMetadata, error) {
	node, err := f.nodesDao.GetFilesMetadataByPathAndRegistryID(ctx, regID, filePath,
		sortByField,
		sortByOrder,
		limit,
		offset,
		search)

	if err != nil {
		return &[]types.FileNodeMetadata{}, fmt.Errorf("failed to get the files "+
			pathFormat, filePath, err)
	}
	return node, nil
}

func (f *FileManager) CountFilesByPath(
	ctx context.Context,
	filePath string,
	regID int64,
) (int64, error) {
	count, err := f.nodesDao.CountByPathAndRegistryID(ctx, regID, filePath)

	if err != nil {
		return -1, fmt.Errorf("failed to get the count of files"+
			pathFormat, filePath, err)
	}

	return count, nil
}

func (f *FileManager) UploadTempFile(
	ctx context.Context,
	rootIdentifier string,
	file multipart.File,
	fileName string,
	fileReader io.Reader,
) (types.FileInfo, string, error) {
	blobContext := f.App.GetBlobsContext(ctx, rootIdentifier)
	tempFileName := uuid.NewString()
	fileInfo, _, err := f.uploadTempFileInternal(ctx, blobContext, rootIdentifier,
		file, fileName, fileReader, tempFileName)
	if err != nil {
		return fileInfo, tempFileName, err
	}
	return fileInfo, tempFileName, nil
}

func (f *FileManager) uploadTempFileInternal(
	ctx context.Context,
	blobContext *Context,
	rootIdentifier string,
	file multipart.File,
	fileName string,
	fileReader io.Reader,
	tempFileName string,
) (types.FileInfo, string, error) {
	tmpPath := path.Join(rootPathString, rootIdentifier, tmp, tempFileName)
	fw, err := blobContext.genericBlobStore.Create(ctx, tmpPath)

	if err != nil {
		log.Error().Msgf("failed to initiate the file upload for file with"+
			" name : %s with error : %s", fileName, err.Error())
		return types.FileInfo{}, tmpPath, fmt.Errorf("failed to initiate the file upload "+
			"for file with name : %s with error : %w", fileName, err)
	}
	defer fw.Close()

	fileInfo, err := blobContext.genericBlobStore.Write(ctx, fw, file, fileReader)
	if err != nil {
		log.Error().Msgf("failed to upload the file on temparary location"+
			" with name : %s with error : %s", fileName, err.Error())
		return types.FileInfo{}, tmpPath, fmt.Errorf("failed to upload the file on temparary "+
			"location with name : %s with error : %w", fileName, err)
	}
	return fileInfo, tmpPath, nil
}

func (f *FileManager) DownloadTempFile(
	ctx context.Context,
	fileSize int64,
	fileName string,
	rootIdentifier string,
) (fileReader *storage.FileReader, size int64, err error) {
	tmpPath := path.Join(rootPathString, rootIdentifier, tmp, fileName)
	blobContext := f.App.GetBlobsContext(ctx, rootIdentifier)
	reader, err := blobContext.genericBlobStore.GetWithNoRedirect(ctx, tmpPath, fileSize)
	if err != nil {
		return nil, 0, fmt.Errorf(failedToGetFile, tmpPath, err)
	}

	return reader, fileSize, nil
}

func (f *FileManager) MoveTempFile(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	rootIdentifier string,
	fileInfo types.FileInfo,
	tempFileName string,
) error {
	// uploading the file to temporary path in file storage
	blobContext := f.App.GetBlobsContext(ctx, rootIdentifier)
	tmpPath := path.Join(rootPathString, rootIdentifier, tmp, tempFileName)

	err := f.moveFile(ctx, rootIdentifier, fileInfo, blobContext, tmpPath)
	if err != nil {
		return err
	}

	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo)
	if err != nil {
		return err
	}

	// Emit blob create event
	if created {
		event.ReportEventAsync(ctx, rootIdentifier, f.reporter, event.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.App.Config)
	}
	return nil
}
