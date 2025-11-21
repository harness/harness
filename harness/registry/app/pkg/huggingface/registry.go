// Copyright 2023 Harness, Inc.
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

package huggingface

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	huggingfacetype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/storage"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type Registry interface {
	pkg.Artifact

	ValidateYaml(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
		headers *commons.ResponseHeaders, response *huggingfacetype.ValidateYamlResponse, err error)

	PreUpload(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
		headers *commons.ResponseHeaders, response *huggingfacetype.PreUploadResponse, err error)

	RevisionInfo(ctx context.Context, info huggingfacetype.ArtifactInfo, queryParams map[string][]string) (
		headers *commons.ResponseHeaders, response *huggingfacetype.RevisionInfoResponse, err error)

	LfsInfo(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser, token string) (
		headers *commons.ResponseHeaders, response *huggingfacetype.LfsInfoResponse, err error)

	LfsUpload(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
		headers *commons.ResponseHeaders, response *huggingfacetype.LfsUploadResponse, err error)

	LfsVerify(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
		headers *commons.ResponseHeaders, response *huggingfacetype.LfsVerifyResponse, err error)

	CommitRevision(ctx context.Context, info huggingfacetype.ArtifactInfo, body io.ReadCloser) (
		headers *commons.ResponseHeaders, response *huggingfacetype.CommitRevisionResponse, err error)

	HeadFile(ctx context.Context, info huggingfacetype.ArtifactInfo, fileName string) (
		responseHeaders *commons.ResponseHeaders, err error)

	DownloadFile(ctx context.Context, info huggingfacetype.ArtifactInfo, fileName string) (
		responseHeaders *commons.ResponseHeaders, body *storage.FileReader, redirectURL string, err error)
}
