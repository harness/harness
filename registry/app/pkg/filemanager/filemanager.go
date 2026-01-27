// Copyright 2023 Harness, Inc.
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
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/events/replication"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const (
	rootPathString = "/"
	nodeLimit      = 1000
	pathFormat     = "for path: %s, with error %w"
)

// FileManager defines the interface for file management operations.

func NewFileManager(
	registryDao store.RegistryRepository, genericBlobDao store.GenericBlobRepository,
	nodesDao store.NodesRepository, tx dbtx.Transactor,
	config *gitnesstypes.Config, storageService *storage.Service,
	bucketService docker.BucketService, replicationReporter replication.Reporter,
	blobCreationDBHook storage.BlobCreationDBHook,
) FileManager {
	return &fileManager{
		registryDao:         registryDao,
		genericBlobDao:      genericBlobDao,
		nodesDao:            nodesDao,
		tx:                  tx,
		config:              config,
		storageService:      storageService,
		bucketService:       bucketService,
		replicationReporter: replicationReporter,
		blobCreationDBHook:  blobCreationDBHook,
	}
}

type fileManager struct {
	config              *gitnesstypes.Config
	storageService      *storage.Service
	registryDao         store.RegistryRepository
	genericBlobDao      store.GenericBlobRepository
	nodesDao            store.NodesRepository
	tx                  dbtx.Transactor
	bucketService       docker.BucketService
	replicationReporter replication.Reporter
	blobCreationDBHook  storage.BlobCreationDBHook
}

// UploadFile - use it to upload file. Inputs can be file or fileReader.
func (f *fileManager) UploadFile(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	rootIdentifier string,
	file multipart.File,
	fileReader io.Reader,
	principalID int64,
) (types.FileInfo, error) {
	blobContext := f.getBlobsContext(ctx, rootIdentifier, "", "", "")
	fileInfo, err := f.uploadAndMove(ctx, blobContext, rootIdentifier, file, fileReader)
	if err != nil {
		return fileInfo, err
	}

	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo, principalID)
	if err != nil {
		return fileInfo, err
	}

	// Emit blob create event
	if created {
		destinations := []replication.CloudLocation{}
		f.replicationReporter.ReportEventAsync(ctx, rootIdentifier, replication.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.config, destinations)
	}
	return fileInfo, nil
}

// GetBlobsContext context constructs the context object for the application. This only be
// called once per request.
func (f *fileManager) getBlobsContext(
	c context.Context, registryIdentifier,
	rootIdentifier, blobID, sha256 string,
) *Context {
	ctx := &Context{Context: c}

	// For reads and Lazy Replication
	if f.bucketService != nil && blobID != "" {
		if result := f.bucketService.GetBlobStore(c, registryIdentifier, rootIdentifier, blobID,
			sha256); result != nil {
			ctx.genericBlobStore = result.GenericStore
			return ctx
		}
	}

	// For default flows
	ctx.genericBlobStore = f.storageService.GenericBlobsStore(c, rootIdentifier)
	return ctx
}

func (f *fileManager) dbSaveFile(
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

	// Saving the nodes
	created := false
	err := f.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		blobID, created, err = f.genericBlobDao.Create(ctx, gb)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to save generic blob in db with "+
				"sha256 : %s, err: %s", fileInfo.Sha256, err.Error())
			return fmt.Errorf("failed to save generic blob"+
				" in db with sha256 : %s, err: %w", fileInfo.Sha256, err)
		}
		err = f.createNodes(ctx, filePath, blobID, regID, createdBy)
		if err != nil {
			return err
		}
		err = f.blobCreationDBHook.AfterBlobCreate(ctx,
			rootParentID,
			types.Digest(gb.Sha1),
			types.Digest(gb.Sha256),
			types.Digest(gb.Sha512),
			types.Digest(gb.MD5),
			gb.Size,
			// TODO(Arvind) This should be provided by storage layer
			-1)
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

func (f *fileManager) SaveNodes(
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

func (f *fileManager) CreateNodesWithoutFileNode(
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

func (f *fileManager) createNodes(
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

func (f *fileManager) SaveNode(
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

func (f *fileManager) CopyNodes(
	ctx context.Context,
	rootParentID int64,
	sourceRegistryID int64,
	targetRegistryID int64,
	sourcePathPrefixes []string,
) error {
	var nodes []types.Node
	for _, sourcePathPrefix := range sourcePathPrefixes {
		n, err := f.nodesDao.GetAllFileNodesByPathPrefixAndRegistryID(ctx, sourceRegistryID, sourcePathPrefix)
		if err != nil {
			return fmt.Errorf("failed to get nodes from source registry for path [%s], err: %w", sourcePathPrefix, err)
		}
		nodes = append(nodes, *n...)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("failed to get nodes from source registry")
	}

	// FIXME: Optimize this flow
	for _, node := range nodes {
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

func (f *fileManager) DownloadFileByPath(
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
	blobContext := f.getBlobsContext(ctx, registryIdentifier, rootIdentifier, blob.ID, blob.Sha256)

	if allowRedirect {
		fileReader, redirectURL, err = blobContext.genericBlobStore.GetGeneric(ctx, blob.Size, node.Name,
			rootIdentifier, blob.Sha256)
	} else {
		fileReader, err = blobContext.genericBlobStore.GetV2NoRedirect(ctx, rootIdentifier, blob.Sha256, blob.Size)
	}

	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get file with digest: %s %w", blob.Sha256, err)
	}

	return fileReader, blob.Size, redirectURL, nil
}

func (f *fileManager) DeleteFile(
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

func (f *fileManager) DeleteLeafNode(
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

func (f *fileManager) GetNode(
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

func (f *fileManager) HeadFile(
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

func (f *fileManager) GetFilePath(
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

func (f *fileManager) FindLatestFilePath(
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

func (f *fileManager) GetFileMetadata(ctx context.Context, regID int64, filePath string) (types.FileInfo, error) {
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

func (f *fileManager) GetFilesMetadata(
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

func (f *fileManager) CountFilesByPath(
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

func (f *fileManager) UploadFileNoDBUpdate(
	ctx context.Context,
	rootIdentifier string,
	file multipart.File,
	fileReader io.Reader,
) (types.FileInfo, error) {
	blobContext := f.getBlobsContext(ctx, rootIdentifier, "", "", "")
	fileInfo, err := f.uploadAndMove(ctx, blobContext, rootIdentifier, file, fileReader)
	if err != nil {
		return fileInfo, err
	}
	return fileInfo, nil
}

func (f *fileManager) HeadByDigest(ctx context.Context, rootIdentifier string, info types.FileInfo) (
	bool,
	int64,
	error,
) {
	blobContext := f.getBlobsContext(ctx, rootIdentifier, "", "", "")
	size, err := blobContext.genericBlobStore.StatByDigest(ctx, rootIdentifier, info.Sha256)
	if err != nil {
		return false, 0, err
	}
	return true, size, nil
}

func (f *fileManager) uploadAndMove(
	ctx context.Context,
	blobContext *Context,
	rootIdentifier string,
	file multipart.File,
	fileReader io.Reader,
) (types.FileInfo, error) {
	fw, err := blobContext.genericBlobStore.CreateGeneric(ctx, rootIdentifier)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to initiate the file upload for file with error : %s", err.Error())
		return types.FileInfo{}, fmt.Errorf("failed to initiate the file upload "+
			"for file with error : %w", err)
	}
	defer fw.Close()

	fileInfo, err := blobContext.genericBlobStore.Write(ctx, fw, file, fileReader)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to upload the file on temparary location"+
			" with error : %s", err.Error())
		return types.FileInfo{}, fmt.Errorf("failed to upload the file on temporary "+
			"location with with error : %w", err)
	}
	err = fw.PlainCommit(ctx, fileInfo.Sha256)
	if err != nil {
		return types.FileInfo{}, err
	}
	return fileInfo, nil
}

// DownloadTempFile These type of APIs should not be introduced. Difficult to track objects and all updates should be
// done inside nodes tables owned by filemanager. If the object is not there, it is supposed to be GCed.
func (f *fileManager) DownloadFileByDigest(
	ctx context.Context,
	rootIdentifier string,
	fileInfo types.FileInfo,
) (fileReader *storage.FileReader, err error) {
	blobContext := f.getBlobsContext(ctx, "", rootIdentifier, "", "")
	reader, err := blobContext.genericBlobStore.GetV2NoRedirect(ctx, rootIdentifier, fileInfo.Sha256, fileInfo.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get file with digest: %s %w", fileInfo.Sha256, err)
	}

	return reader, nil
}

func (f *fileManager) PostFileUpload(
	ctx context.Context,
	filePath string,
	regID int64,
	rootParentID int64,
	rootIdentifier string,
	fileInfo types.FileInfo,
	principalID int64,
) error {
	blobID, created, err := f.dbSaveFile(ctx, filePath, regID, rootParentID, fileInfo, principalID)
	if err != nil {
		return err
	}

	// Emit blob create event
	if created {
		destinations := []replication.CloudLocation{}
		f.replicationReporter.ReportEventAsync(ctx, rootIdentifier, replication.BlobCreate, 0, blobID, fileInfo.Sha256,
			f.config, destinations)
	}
	return nil
}
