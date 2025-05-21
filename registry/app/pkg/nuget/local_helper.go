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

package nuget

import (
	"context"
	"io"
	"strings"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
)

type LocalRegistryHelper interface {
	FileExists(ctx context.Context, info nugettype.ArtifactInfo) bool
	DownloadFile(ctx context.Context, info nugettype.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)
	UploadPackageFile(
		ctx context.Context,
		info nugettype.ArtifactInfo,
		fileReader io.ReadCloser,
	) (*commons.ResponseHeaders, string, error)
	DeletePackage(ctx context.Context, info nugettype.ArtifactInfo) error
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

func (h *localRegistryHelper) FileExists(ctx context.Context, info nugettype.ArtifactInfo) bool {
	return h.localBase.Exists(ctx, info.ArtifactInfo,
		pkg.JoinWithSeparator("/", info.Image, info.Version, info.Filename))
}

func (h *localRegistryHelper) DownloadFile(ctx context.Context, info nugettype.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	error,
) {
	return h.localBase.Download(ctx, info.ArtifactInfo, info.Version, info.Filename)
}

func (h *localRegistryHelper) DeletePackage(ctx context.Context, info nugettype.ArtifactInfo) error {
	return h.localBase.DeleteVersion(ctx, info)
}

func (h *localRegistryHelper) UploadPackageFile(
	ctx context.Context,
	info nugettype.ArtifactInfo,
	fileReader io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	fileExtension := DependencyFile
	if strings.HasSuffix(info.Filename, SymbolsPackageExtension) {
		fileExtension = SymbolsFile
	}
	return h.localRegistry.UploadPackage(ctx, info, fileReader, fileExtension)
}
