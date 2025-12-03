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
	"io"

	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
)

type LocalRegistryHelper interface {
	FileExists(ctx context.Context, info generic.ArtifactInfo) (*commons.ResponseHeaders, error)
	DownloadFile(ctx context.Context, info generic.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)
	PutFile(
		ctx context.Context,
		info generic.ArtifactInfo,
		fileReader io.ReadCloser,
		contentType string,
	) (headers *commons.ResponseHeaders, sha256 string, err error)

	DeleteFile(ctx context.Context, info generic.ArtifactInfo) (*commons.ResponseHeaders, error)
}

type localRegistryHelper struct {
	localRegistry LocalRegistry
	localBase     base.LocalBase
}

func NewLocalRegistryHelper(localRegistry LocalRegistry, localBase base.LocalBase) LocalRegistryHelper {
	return &localRegistryHelper{
		localRegistry: localRegistry,
		localBase:     localBase,
	}
}

func (h *localRegistryHelper) FileExists(ctx context.Context, info generic.ArtifactInfo) (
	*commons.ResponseHeaders,
	error,
) {
	headers, err := h.localBase.ExistsE(ctx, info, info.FilePath)
	return headers, err
}

func (h *localRegistryHelper) DownloadFile(ctx context.Context, info generic.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	error,
) {
	return h.localBase.Download(ctx, info.ArtifactInfo, info.Version, info.FilePath)
}

func (h *localRegistryHelper) PutFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	fileReader io.ReadCloser,
	contentType string,
) (*commons.ResponseHeaders, string, error) {
	responseHeaders, sha256, err := h.localRegistry.PutFile(ctx, info, fileReader, contentType)
	return responseHeaders, sha256, err
}

func (h *localRegistryHelper) DeleteFile(ctx context.Context, info generic.ArtifactInfo) (
	*commons.ResponseHeaders,
	error,
) {
	return h.localRegistry.DeleteFile(ctx, info)
}
