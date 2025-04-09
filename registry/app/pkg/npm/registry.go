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

	npm3 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	pkg.Artifact
	UploadPackageFile(
		ctx context.Context,
		info npm.ArtifactInfo,
		file io.ReadCloser,
	) (*commons.ResponseHeaders, string, error)

	DownloadPackageFile(ctx context.Context, info npm.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)

	GetPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (npm3.PackageMetadata, error)

	HeadPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (bool, error)

	ListTags(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error)

	AddTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error)

	DeleteTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error)

	DeletePackage(ctx context.Context, info npm.ArtifactInfo) error
	DeleteVersion(ctx context.Context, info npm.ArtifactInfo) error
}
