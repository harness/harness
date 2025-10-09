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

package npm

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
)

type LocalRegistryHelper interface {
	FileExists(ctx context.Context, info npm.ArtifactInfo) bool
	DownloadFile(ctx context.Context, info npm.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)
	UploadPackageFile(
		ctx context.Context,
		info npm.ArtifactInfo,
		file io.ReadCloser,
	) (*commons.ResponseHeaders, string, error)
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

func (h *localRegistryHelper) FileExists(ctx context.Context, info npm.ArtifactInfo) bool {
	return h.localBase.Exists(ctx, info.ArtifactInfo,
		pkg.JoinWithSeparator("/", info.Image, info.Version, info.Filename))
}

func (h *localRegistryHelper) DownloadFile(ctx context.Context, info npm.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	error,
) {
	return h.localBase.Download(ctx, info.ArtifactInfo, info.Version, info.Filename)
}

func (h *localRegistryHelper) UploadPackageFile(
	ctx context.Context,
	info npm.ArtifactInfo,
	file io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	return h.localRegistry.UploadPackageFileWithoutParsing(ctx, info, file)
}
