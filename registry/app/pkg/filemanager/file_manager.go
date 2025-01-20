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

	"github.com/harness/gitness/registry/app/pkg"
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
)

func NewFileManager(app *App, registryDao store.RegistryRepository, genericBlobDao store.GenericBlobRepository,
	nodesDao store.NodesRepository,
	tx dbtx.Transactor,
) FileManager {
	return FileManager{
		App:            app,
		registryDao:    registryDao,
		genericBlobDao: genericBlobDao,
		nodesDao:       nodesDao,
		tx:             tx,
	}
}

type FileManager struct {
	App            *App
	registryDao    store.RegistryRepository
	genericBlobDao store.GenericBlobRepository
	nodesDao       store.NodesRepository
	tx             dbtx.Transactor
}

func (f *FileManager) UploadFile(
	ctx context.Context,
	filePath string,
	regName string,
	regID int64,
	rootParentID int64,
	rootIdentifier string,
	file multipart.File,
	fileReader io.Reader,
	filename string,
) (pkg.FileInfo, error) {
	// uploading the file to temporary path in file storage
	blobContext := f.App.GetBlobsContext(ctx, regName, rootIdentifier)
	pathUUID := uuid.NewString()
	tmpPath := path.Join(rootPathString, rootIdentifier, tmp, pathUUID)
	fw, err := blobContext.genericBlobStore.Create(ctx, tmpPath)

	if err != nil {
		log.Error().Msgf("failed to initiate the file upload for file with"+
			" name : %s with error : %s", filename, err.Error())
		return pkg.FileInfo{}, fmt.Errorf("failed to initiate the file upload "+
			"for file with name : %s with error : %w", filename, err)
	}
	defer fw.Close()

	fileInfo, err := blobContext.genericBlobStore.Write(ctx, fw, file, fileReader)
	if err != nil {
		log.Error().Msgf("failed to upload the file on temparary location"+
			" with name : %s with error : %s", filename, err.Error())
		return pkg.FileInfo{}, fmt.Errorf("failed to upload the file on temparary "+
			"location with name : %s with error : %w", filename, err)
	}
	fileInfo.Filename = filename

	// Moving the file to permanent path in file storage
	fileStoragePath := path.Join(rootPathString, rootIdentifier, files, fileInfo.Sha256)
	err = blobContext.genericBlobStore.Move(ctx, tmpPath, fileStoragePath)

	if err != nil {
		log.Error().Msgf("failed to Move the file on permanent location "+
			"with name : %s with error : %s", filename, err.Error())
		return pkg.FileInfo{}, fmt.Errorf("failed to Move the file on permanent"+
			" location with name : %s with error : %w", filename, err)
	}

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
	err = f.genericBlobDao.Create(ctx, gb)
	if err != nil {
		log.Error().Msgf("failed to save generic blob in db with "+
			"sha256 : %s, err: %s", fileInfo.Sha256, err.Error())
		return pkg.FileInfo{}, fmt.Errorf("failed to save generic blob"+
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
			"path : %s, err: %s", filename, filePath, err)
		return pkg.FileInfo{}, fmt.Errorf("failed to save nodes for"+
			" file : %s, with path : %s, err: %w", filename, filePath, err)
	}
	return fileInfo, nil
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

func (f *FileManager) SaveNode(ctx context.Context, filePath string, blobID string, regID int64, segment string,
	parentID string, nodePath string, isFile bool) (string, error) {
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
			"for path := %s", segment, filePath)
	}
	return node.ID, nil
}

func (f *FileManager) DownloadFile(
	ctx context.Context,
	filePath string,
	regInfo types.Registry,
	rootIdentifier string,
) (fileReader *storage.FileReader, size int64, err error) {
	node, err := f.nodesDao.GetByPathAndRegistryId(ctx, regInfo.ID, filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get the node for path: %s, "+
			"with registry: %s, with error %s", filePath, regInfo.Name, err)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get the blob for path: %s, "+
			"with blob id: %s, with error %s", filePath, blob.ID, err)
	}

	completeFilaPath := path.Join(rootPathString + rootIdentifier + rootPathString + files + rootPathString + blob.Sha256)
	//
	blobContext := f.App.GetBlobsContext(ctx, regInfo.Name, rootIdentifier)
	reader, err := blobContext.genericBlobStore.Get(ctx, completeFilaPath, blob.Size)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get the file for path: %s, "+
			" with error %w", completeFilaPath, err)
	}
	return reader, blob.Size, nil
}

func (f *FileManager) DeleteFile(
	ctx context.Context,
	filePath string,
	regID int,
) error {
	log.Ctx(ctx).Info().Msgf("%s%d", filePath, regID)
	return nil
}

func (f *FileManager) HeadFile(
	ctx context.Context,
	filePath string,
	regID int64,
) (string, error) {
	node, err := f.nodesDao.GetByPathAndRegistryId(ctx, regID, filePath)

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
) (pkg.FileInfo, error) {
	node, err := f.nodesDao.GetByPathAndRegistryId(ctx, regID, filePath)

	if err != nil {
		return pkg.FileInfo{}, fmt.Errorf("failed to get the node path mapping "+
			"for path: %s, with error %w", filePath, err)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return pkg.FileInfo{}, fmt.Errorf("failed to get the blob for path: %s, "+
			"with blob id: %s, with error %s", filePath, node.BlobID, err)
	}
	return pkg.FileInfo{
		Sha1:     blob.Sha1,
		Size:     blob.Size,
		Sha256:   blob.Sha256,
		Sha512:   blob.Sha512,
		MD5:      blob.MD5,
		Filename: node.Name,
	}, nil
}
