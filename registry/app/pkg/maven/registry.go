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

package maven

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	Artifact

	HeadArtifact(ctx context.Context, artInfo pkg.MavenArtifactInfo) (
		responseHeaders *commons.ResponseHeaders, errs []error)

	GetArtifact(ctx context.Context, artInfo pkg.MavenArtifactInfo) (
		responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser, errs []error)

	PutArtifact(ctx context.Context, artInfo pkg.MavenArtifactInfo, fileReader io.Reader) (
		responseHeaders *commons.ResponseHeaders, errs []error)
}
