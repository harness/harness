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

package gopackage

import (
	"context"
	"io"
	"strings"

	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/app/storage"
)

type LocalRegistryHelper interface {
	FileExists(ctx context.Context, info gopackage.ArtifactInfo, filepath string) bool
	RegeneratePackageIndex(ctx context.Context, info gopackage.ArtifactInfo)
	DownloadFile(ctx context.Context, info gopackage.ArtifactInfo) (
		*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error,
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

func (h *localRegistryHelper) FileExists(
	ctx context.Context, info gopackage.ArtifactInfo, filepath string,
) bool {
	return h.localBase.Exists(ctx, info.ArtifactInfo, strings.TrimLeft(filepath, "/"))
}

func (h *localRegistryHelper) RegeneratePackageIndex(
	ctx context.Context, info gopackage.ArtifactInfo,
) {
	h.postProcessingReporter.BuildPackageIndex(ctx, info.RegistryID, info.Image)
}

func (h *localRegistryHelper) DownloadFile(ctx context.Context, info gopackage.ArtifactInfo) (
	*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error,
) {
	return h.localRegistry.DownloadPackageFile(ctx, info)
}
