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

package cargo

import (
	"context"
	"io"
	"strings"

	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
)

type LocalRegistryHelper interface {
	FileExists(ctx context.Context, info cargotype.ArtifactInfo) bool
	DownloadFile(ctx context.Context, info cargotype.ArtifactInfo) (
		*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error,
	)
	MoveTempFile(
		ctx context.Context,
		info *cargotype.ArtifactInfo,
		fileInfo types.FileInfo,
		metadata *cargometadata.VersionMetadata,
	) (*commons.ResponseHeaders, string, int64, bool, error)
	UpdatePackageIndex(
		ctx context.Context, info cargotype.ArtifactInfo,
	)
}

type localRegistryHelper struct {
	localRegistry          LocalRegistry
	localBase              base.LocalBase
	postProcessingReporter *asyncprocessing.Reporter
}

func NewLocalRegistryHelper(
	localRegistry LocalRegistry, localBase base.LocalBase,
	postProcessingReporter *asyncprocessing.Reporter,
) LocalRegistryHelper {
	return &localRegistryHelper{
		localRegistry:          localRegistry,
		localBase:              localBase,
		postProcessingReporter: postProcessingReporter,
	}
}

func (h *localRegistryHelper) FileExists(ctx context.Context, info cargotype.ArtifactInfo) bool {
	return h.localBase.Exists(ctx, info.ArtifactInfo, strings.TrimLeft(getCrateFilePath(info.Image, info.Version), "/"))
}

func (h *localRegistryHelper) DownloadFile(ctx context.Context, info cargotype.ArtifactInfo) (
	*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error,
) {
	return h.localRegistry.DownloadPackage(ctx, info)
}

func (h *localRegistryHelper) UpdatePackageIndex(
	ctx context.Context, info cargotype.ArtifactInfo,
) {
	h.postProcessingReporter.BuildPackageIndex(ctx, info.RegistryID, info.Image)
}

func (h *localRegistryHelper) MoveTempFile(
	ctx context.Context,
	info *cargotype.ArtifactInfo,
	fileInfo types.FileInfo,
	metadata *cargometadata.VersionMetadata,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	return h.localBase.UpdateFileManagerAndCreateArtifact(ctx, info.ArtifactInfo, info.Version,
		getCrateFilePath(info.Image, info.Version), &cargometadata.VersionMetadataDB{
			VersionMetadata: *metadata,
		}, fileInfo, false)
}
