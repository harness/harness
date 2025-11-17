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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/event"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	gitnesstypes "github.com/harness/gitness/types"

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
	registryDao store.RegistryRepository, genericBlobDao store.GenericBlobRepository,
	nodesDao store.NodesRepository, tx dbtx.Transactor,
	reporter event.Reporter, config *gitnesstypes.Config,
	storageService *storage.Service,
	bucketService docker.BucketService,
) FileManager {
	return FileManager{
		registryDao:    registryDao,
		genericBlobDao: genericBlobDao,
		nodesDao:       nodesDao,
		tx:             tx,
		reporter:       reporter,
		config:         config,
		storageService: storageService,
		bucketService:  bucketService,
	}
}

type FileManager struct {
	config         *gitnesstypes.Config
	storageService *storage.Service
	registryDao    store.RegistryRepository
	genericBlobDao store.GenericBlobRepository
	nodesDao       store.NodesRepository
	tx             dbtx.Transactor
	reporter       event.Reporter
	bucketService  docker.BucketService
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
	principalID int64,
) (types.FileInfo, error) {
	// uploading the file to temporary path in file storage
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
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

	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo, principalID)
	if err != nil {
		return fileInfo, err
	}

	// Emit blob create event
	if created {
		destinations := []event.CloudLocation{}
		event.ReportEventAsync(ctx, rootIdentifier, f.reporter, event.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.config, destinations)
	}
	return fileInfo, nil
}

// GetBlobsContext context constructs the context object for the application. This only be
// called once per request.
func (f *FileManager) GetBlobsContext(
	c context.Context, registryIdentifier,
	rootIdentifier, blobID, sha256 string,
) *Context {
	ctx := &Context{Context: c}

	if f.bucketService != nil && blobID != "" {
		if result := f.bucketService.GetBlobStore(c, registryIdentifier, rootIdentifier, blobID,
			sha256); result != nil {
			ctx.genericBlobStore = result.GenericStore
			return ctx
		}
	}
	// use blob store from the default bucket
	ctx.genericBlobStore = f.storageService.GenericBlobsStore(rootIdentifier)
	return ctx
}

func (f *FileManager) dbSaveFile(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	fileInfo types.FileInfo,
	createdBy int64,
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
		CreatedBy:    createdBy,
	}
	created, err := f.genericBlobDao.Create(ctx, gb)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to save generic blob in db with "+
			"sha256 : %s, err: %s", fileInfo.Sha256, err.Error())
		return "", false, fmt.Errorf("failed to save generic blob"+
			" in db with sha256 : %s, err: %w", fileInfo.Sha256, err)
	}
	blobID = gb.ID
	// Saving the nodes
	err = f.tx.WithTx(ctx, func(ctx context.Context) error {
		err = f.createNodes(ctx, filePath, blobID, regID, createdBy)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to save nodes for file : %s, with "+
			"path : %s, err: %s", fileInfo.Filename, filePath, err)
		return "", false, fmt.Errorf("failed to save nodes for"+
			" file : %s, with path : %s, err: %w", fileInfo.Filename, filePath, err)
	}
	return blobID, created, nil
}

func (f *FileManager) SaveNodes(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	createdBy int64,
	sha256 string,
) error {
	gb, err := f.genericBlobDao.FindBySha256AndRootParentID(ctx, sha256, rootParentID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to find generic blob in db with "+
			"sha256 : %s, err: %s", sha256, err.Error())
		return fmt.Errorf("failed to find generic blob"+
			" in db with sha256 : %s, err: %w", sha256, err)
	}
	// Saving the nodes
	err = f.createNodes(ctx, filePath, gb.ID, regID, createdBy)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to save nodes for file with "+
			"path : %s, err: %s", filePath, err)
		return fmt.Errorf("failed to save nodes for file with path : %s, err: %w", filePath, err)
	}
	return nil
}

func (f *FileManager) SaveNodesTx(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	createdBy int64,
	sha256 string,
) error {
	return f.tx.WithTx(ctx, func(ctx context.Context) error {
		return f.SaveNodes(ctx, filePath, regID, rootParentID, createdBy, sha256)
	})
}

func (f *FileManager) CreateNodesWithoutFileNode(
	ctx context.Context,
	path string,
	regID int64,
	principalID int64,
) error {
	segments := strings.Split(path, rootPathString)
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

		nodeID, err = f.SaveNode(ctx, path, "", regID, segment,
			parentID, nodePath, false, principalID)
		if err != nil {
			return err
		}
		parentID = nodeID
	}
	return nil
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
		log.Ctx(ctx).Error().Msgf("failed to Move the file on permanent location "+
			"with name : %s with error : %s", fileInfo.Filename, err.Error())
		return fmt.Errorf("failed to Move the file on permanent"+
			" location with name : %s with error : %w", fileInfo.Filename, err)
	}
	return nil
}

func (f *FileManager) createNodes(
	ctx context.Context,
	filePath string,
	blobID string,
	regID int64,
	principalID int64,
) error {
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
				parentID, nodePath, true, principalID)
			if err != nil {
				return err
			}
		} else {
			nodeID, err = f.SaveNode(ctx, filePath, "", regID, segment,
				parentID, nodePath, false, principalID)
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
	parentID string, nodePath string, isFile bool, createdBy int64,
) (string, error) {
	node := &types.Node{
		Name:         segment,
		RegistryID:   regID,
		ParentNodeID: parentID,
		IsFile:       isFile,
		NodePath:     nodePath,
		BlobID:       blobID,
		CreatedBy:    createdBy,
	}
	err := f.nodesDao.Create(ctx, node)
	if err != nil {
		return "", fmt.Errorf("failed to create the node: %s, "+
			"for path := %s, %w", segment, filePath, err)
	}
	return node.ID, nil
}

func (f *FileManager) CopyNodes(
	ctx context.Context,
	rootParentID int64,
	sourceRegistryID int64,
	targetRegistryID int64,
	sourcePathPrefix string,
) error {
	nodes, err := f.nodesDao.GetAllFileNodesByPathPrefixAndRegistryID(ctx, sourceRegistryID, sourcePathPrefix)
	if err != nil || nodes == nil || len(*nodes) == 0 {
		return fmt.Errorf("failed to get nodes from source registry: %w", err)
	}

	// FIXME: Optimize this flow
	for _, node := range *nodes {
		blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)
		if err != nil {
			return fmt.Errorf("failed to get blob: %s, %w", node.BlobID, err)
		}
		err = f.SaveNodes(ctx, node.NodePath, targetRegistryID, rootParentID, node.CreatedBy, blob.Sha256)
		if err != nil {
			return fmt.Errorf("failed to save nodes: %w", err)
		}
	}
	return nil
}

func (f *FileManager) DownloadFile(
	ctx context.Context,
	filePath string,
	registryID int64,
	registryIdentifier string,
	rootIdentifier string,
	allowRedirect bool,
) (fileReader *storage.FileReader, size int64, redirectURL string, err error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, registryID, filePath)
	if err != nil {
		return nil, 0, "", usererror.NotFoundf("file not found for registry [%s], path: [%s]", registryIdentifier,
			filePath)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return nil, 0, "", usererror.NotFoundf("failed to get the blob for path: %s, "+
			"with blob id: %s, with error %s", filePath, node.BlobID, err)
	}
	completeFilaPath := path.Join(rootPathString + rootIdentifier + rootPathString + files + rootPathString + blob.Sha256)
	blobContext := f.GetBlobsContext(ctx, registryIdentifier, rootIdentifier, blob.ID, blob.Sha256)

	if allowRedirect {
		fileReader, redirectURL, err = blobContext.genericBlobStore.Get(ctx, completeFilaPath, blob.Size, node.Name)
	} else {
		fileReader, err = blobContext.genericBlobStore.GetWithNoRedirect(ctx, completeFilaPath, blob.Size)
	}

	if err != nil {
		return nil, 0, "", fmt.Errorf(failedToGetFile, completeFilaPath, err)
	}

	return fileReader, blob.Size, redirectURL, nil
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

func (f *FileManager) DeleteLeafNode(
	ctx context.Context,
	regID int64,
	filePath string,
) error {
	if len(filePath) > 0 && !strings.HasPrefix(filePath, rootPathString) {
		filePath = rootPathString + filePath
	}
	err := f.nodesDao.DeleteByLeafNodePathAndRegistryID(ctx, filePath, regID)
	if err != nil {
		return fmt.Errorf("failed to delete file for path: %s, with error: %w", filePath, err)
	}
	return nil
}

func (f *FileManager) GetNode(
	ctx context.Context,
	regID int64,
	filePath string,
) (*types.Node, error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, regID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to delete file for path: %s, with error: %w", filePath, err)
	}
	return node, nil
}

func (f *FileManager) HeadFile(
	ctx context.Context,
	filePath string,
	regID int64,
) (sha256 string, size int64, err error) {
	node, err := f.nodesDao.GetByPathAndRegistryID(ctx, regID, filePath)

	if err != nil {
		return "", 0, fmt.Errorf("failed to get the node path mapping for path: %s, "+
			"with error %w", filePath, err)
	}
	blob, err := f.genericBlobDao.FindByID(ctx, node.BlobID)

	if err != nil {
		return "", 0, fmt.Errorf("failed to get the blob for path: %s, with blob id: %s,"+
			" with error %w", filePath, node.BlobID, err)
	}
	return blob.Sha256, blob.Size, nil
}

func (f *FileManager) HeadSHA256(
	ctx context.Context,
	sha256 string,
	regID int64,
	rootParentID int64,
) (string, error) {
	blob, err := f.genericBlobDao.FindBySha256AndRootParentID(ctx, sha256, rootParentID)

	if blob == nil || err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get the blob for sha256: %s, with root parent id: %d, with error %v",
			sha256, rootParentID, err)
		return "", fmt.Errorf("failed to get the blob for sha256: %s, with root parent id: %d, with error %w", sha256,
			rootParentID, err)
	}

	node, err := f.nodesDao.GetByBlobIDAndRegistryID(ctx, blob.ID, regID)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get the node for blob id: %s, with registry id: %d, with error %v",
			blob.ID, regID, err)
		return "", fmt.Errorf("failed to get the node for blob id: %s, with registry id: %d, with error %w", blob.ID,
			regID, err)
	}

	return node.NodePath, nil
}

func (f *FileManager) FindLatestFilePath(
	ctx context.Context, registryID int64,
	filepathPrefix, filename string,
) (string, error) {
	fileNode, err := f.nodesDao.FindByPathAndRegistryID(ctx, registryID, filepathPrefix, filename)
	if err != nil {
		return "", fmt.Errorf("failed to get the node for path: %s, file name: %s, with registry id: %d,"+
			" with error %w", filepathPrefix, filename, registryID, err)
	}
	return fileNode.NodePath, nil
}

func (f *FileManager) HeadBlob(
	ctx context.Context,
	sha256 string,
	rootParentID int64,
) (string, error) {
	blob, err := f.genericBlobDao.FindBySha256AndRootParentID(ctx, sha256, rootParentID)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get the blob for sha256: %s, with root parent id: %d, with error %v",
			sha256, rootParentID, err)
		return "", fmt.Errorf("failed to get the blob for sha256: %s, with root parent id: %d, with error %w", sha256,
			rootParentID, err)
	}
	return blob.ID, nil
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
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
	tempFileName := uuid.NewString()
	fileInfo, _, err := f.uploadTempFileInternal(ctx, blobContext, rootIdentifier,
		file, fileName, fileReader, tempFileName)
	if err != nil {
		return fileInfo, tempFileName, err
	}
	return fileInfo, tempFileName, nil
}

func (f *FileManager) UploadTempFileToPath(
	ctx context.Context,
	rootIdentifier string,
	file multipart.File,
	fileName string,
	tempFileName string,
	fileReader io.Reader,
) (types.FileInfo, string, error) {
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
	fileInfo, _, err := f.uploadTempFileInternal(ctx, blobContext, rootIdentifier,
		file, fileName, fileReader, tempFileName)
	if err != nil {
		return fileInfo, tempFileName, err
	}
	return fileInfo, tempFileName, nil
}

func (f *FileManager) FileExists(
	ctx context.Context,
	rootIdentifier string,
	filePath string,
) (bool, int64, error) {
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
	size, err := blobContext.genericBlobStore.Stat(ctx, filePath)
	if err != nil {
		return false, -1, err
	}
	return size > 0, size, nil
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
		log.Ctx(ctx).Error().Msgf("failed to initiate the file upload for file with"+
			" name : %s with error : %s", fileName, err.Error())
		return types.FileInfo{}, tmpPath, fmt.Errorf("failed to initiate the file upload "+
			"for file with name : %s with error : %w", fileName, err)
	}
	defer fw.Close()

	fileInfo, err := blobContext.genericBlobStore.Write(ctx, fw, file, fileReader)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to upload the file on temparary location"+
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
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
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
	principalID int64,
) error {
	// uploading the file to temporary path in file storage
	blobContext := f.GetBlobsContext(ctx, rootIdentifier, "", "", "")
	tmpPath := path.Join(rootPathString, rootIdentifier, tmp, tempFileName)
	err := f.moveFile(ctx, rootIdentifier, fileInfo, blobContext, tmpPath)
	if err != nil {
		return err
	}

	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo, principalID)
	if err != nil {
		return err
	}

	// Emit blob create event
	if created {
		destinations := []event.CloudLocation{}
		event.ReportEventAsync(ctx, rootIdentifier, f.reporter, event.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.config, destinations)
	}
	return nil
}

func (f *FileManager) GetFileMetadataByPathAndRegistryID(
	ctx context.Context,
	registryID int64,
	path string,
) (*types.FileNodeMetadata, error) {
	metadata, err := f.nodesDao.GetFileMetadataByPathAndRegistryID(ctx, registryID, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get [%s] for registry [%d], error: %w", path, registryID, err)
	}
	return metadata, nil
}
