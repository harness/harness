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
	"io"
	"mime/multipart"

	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
)

type FileManager interface {
	UploadFile(
		ctx context.Context,
		filePath string,
		regID int64,
		rootParentID int64,
		rootIdentifier string,
		file multipart.File,
		fileReader io.Reader,
		principalID int64,
	) (types.FileInfo, error)

	DownloadFileByPath(
		ctx context.Context,
		filePath string,
		registryID int64,
		registryIdentifier string,
		rootIdentifier string,
		allowRedirect bool,
	) (fileReader *storage.FileReader, size int64, redirectURL string, err error)

	DownloadFileByDigest(ctx context.Context, rootIdentifier string, info types.FileInfo) (
		fileReader *storage.FileReader,
		err error,
	)

	DeleteFile(
		ctx context.Context,
		regID int64,
		filePath string,
	) error

	HeadFile(
		ctx context.Context,
		filePath string,
		regID int64,
	) (sha256 string, size int64, err error)

	// CopyNodes - Used for duplicating the registry/package/version/file
	CopyNodes(
		ctx context.Context,
		rootParentID int64,
		sourceRegistryID int64,
		targetRegistryID int64,
		sourcePathPrefixes []string,
	) error

	FindLatestFilePath(
		ctx context.Context, registryID int64,
		filepathPrefix, filename string,
	) (string, error)

	GetFilesMetadata(
		ctx context.Context,
		filePath string,
		regID int64,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
	) (*[]types.FileNodeMetadata, error)

	CountFilesByPath(
		ctx context.Context,
		filePath string,
		regID int64,
	) (int64, error)

	PostFileUpload(
		ctx context.Context,
		filePath string,
		regID int64,
		rootParentID int64,
		rootIdentifier string,
		fileInfo types.FileInfo,
		principalID int64,
	) error

	HeadByDigest(ctx context.Context, rootIdentifier string, filePath types.FileInfo) (bool, int64, error)

	// SaveNodes TODO: Need to understand the usecase OR deprecate this function
	SaveNodes(
		ctx context.Context,
		filePath string,
		regID int64,
		rootParentID int64,
		createdBy int64,
		sha256 string,
	) error

	// CreateNodesWithoutFileNode
	// TODO: Need to understand the usecase OR deprecate this function. Currently being used outside gitness.
	CreateNodesWithoutFileNode(
		ctx context.Context,
		path string,
		regID int64,
		principalID int64,
	) error

	// SaveNode TODO: Need to understand the usecase OR deprecate this function. Currently being used outside gitness.
	SaveNode(
		ctx context.Context, filePath string, blobID string, regID int64, segment string,
		parentID string, nodePath string, isFile bool, createdBy int64,
	) (string, error)

	// GetFilePath
	// TODO: Update/Remove this function. This is buggy as multiple file paths can contains same sha256 in a registry
	GetFilePath(
		ctx context.Context,
		sha256 string,
		regID int64,
		rootParentID int64,
	) (string, error)

	// DeleteLeafNode TODO: Need to understand the usecase OR deprecate this function
	DeleteLeafNode(
		ctx context.Context,
		regID int64,
		filePath string,
	) error

	// GetNode
	// TODO: Merge both GetNode and GetFileMetadata functions and return types.FileNodeMetadata with ID
	GetNode(
		ctx context.Context,
		regID int64,
		filePath string,
	) (*types.Node, error)

	// GetFileMetadata
	// TODO: Merge both GetNode and GetFileMetadata functions and return types.FileNodeMetadata with ID
	GetFileMetadata(ctx context.Context, regID int64, filePath string) (types.FileInfo, error)

	// UploadFileNoDBUpdate
	// TODO: This is buggy as we should use filesystem to track all files being uploaded.
	UploadFileNoDBUpdate(
		ctx context.Context,
		rootIdentifier string,
		file multipart.File,
		fileReader io.Reader,
	) (types.FileInfo, error)
}
