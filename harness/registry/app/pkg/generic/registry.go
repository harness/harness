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

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	pkg.Artifact

	PutFile(
		ctx context.Context,
		info generic.ArtifactInfo,
		reader io.ReadCloser,
		contentType string,
	) (
		headers *commons.ResponseHeaders,
		sha256 string,
		err error,
	)

	DownloadFile(ctx context.Context, info generic.ArtifactInfo, filePath string) (
		headers *commons.ResponseHeaders,
		fileReader *storage.FileReader,
		readCloser io.ReadCloser,
		redirectURL string,
		err error,
	)

	DeleteFile(ctx context.Context, info generic.ArtifactInfo) (headers *commons.ResponseHeaders, err error)

	HeadFile(ctx context.Context, info generic.ArtifactInfo, filePath string) (
		headers *commons.ResponseHeaders,
		err error,
	)
}
