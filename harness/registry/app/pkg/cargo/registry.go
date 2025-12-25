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

	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	pkg.Artifact
	// Upload package to registry using cargo CLI
	UploadPackage(
		ctx context.Context, info cargotype.ArtifactInfo,
		metadata *cargometadata.VersionMetadata, crateFile io.ReadCloser,
	) (responseHeaders *commons.ResponseHeaders, err error)
	// download package index metadata file
	DownloadPackageIndex(
		ctx context.Context, info cargotype.ArtifactInfo,
		filePath string,
	) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error)
	// regenerate package index
	RegeneratePackageIndex(ctx context.Context, info cargotype.ArtifactInfo,
	) (*commons.ResponseHeaders, error)
	DownloadPackage(
		ctx context.Context, info cargotype.ArtifactInfo,
	) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error)
	UpdateYank(
		ctx context.Context, info cargotype.ArtifactInfo, yank bool,
	) (*commons.ResponseHeaders, error)
}
