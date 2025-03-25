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

package python

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	pkg.Artifact

	GetPackageMetadata(ctx context.Context, info python.ArtifactInfo) (python.PackageMetadata, error)

	UploadPackageFile(
		ctx context.Context,
		info python.ArtifactInfo,
		file multipart.File,
		filename string,
	) (*commons.ResponseHeaders, string, errcode.Error)

	UploadPackageFileReader(
		ctx context.Context,
		info python.ArtifactInfo,
		file io.ReadCloser,
		filename string,
	) (*commons.ResponseHeaders, string, errcode.Error)

	DownloadPackageFile(ctx context.Context, info python.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		io.ReadCloser,
		string,
		[]error,
	)
}
