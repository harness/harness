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

package huggingface

import (
	"context"
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/huggingface"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

// Controller defines the interface for Huggingface package operations.
type Controller interface {
	ValidateYaml(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser) *ValidateYamlResponse
	PreUpload(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser) *PreUploadResponse
	RevisionInfo(ctx context.Context, info hftype.ArtifactInfo, queryParams map[string][]string) *RevisionInfoResponse
	LfsInfo(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser, token string) *LfsInfoResponse
	LfsVerify(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser) *LfsVerifyResponse
	CommitRevision(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser) *CommitRevisionResponse
	LfsUpload(ctx context.Context, info hftype.ArtifactInfo, body io.ReadCloser) *LfsUploadResponse
	HeadFile(ctx context.Context, info hftype.ArtifactInfo, fileName string) *HeadFileResponse
	DownloadFile(ctx context.Context, info hftype.ArtifactInfo, fileName string) *DownloadFileResponse
}

// controller implements the Controller interface.
type controller struct {
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
	local       huggingface.LocalRegistry
}

// NewController creates a new Huggingface controller.
func NewController(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local huggingface.LocalRegistry,
) Controller {
	return &controller{
		proxyStore:  proxyStore,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		fileManager: fileManager,
		tx:          tx,
		urlProvider: urlProvider,
		local:       local,
	}
}
