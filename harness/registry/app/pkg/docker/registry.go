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

package docker

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

/*
Registry defines the capabilities that an artifact registry should have. It should support following methods:

// Manifest - GET, HEAD, PUT, DELETE
// Catalog - GET
// Tag - GET, DELETE
// Blob - GET, HEAD, DELETE
// Blob Upload - GET, HEAD, POST, PATCH, PUT, DELETE

ref: https://github.com/opencontainers/distribution-spec/blob/main/spec.md

Endpoints to support:

| ID      | Method         | API Endpoint                                                   | Success     | Failure
| ------- | -------------- | -------------------------------------------------------------- | ----------- | -----------
| end-1   | `GET`          | `/v2/`                                                         | `200`       | `404`/`401`
| end-2   | `GET` / `HEAD` | `/v2/<name>/blobs/<digest>`                                    | `200`       | `404`
| end-3   | `GET` / `HEAD` | `/v2/<name>/manifests/<reference>`                             | `200`       | `404`
| end-4a  | `POST`         | `/v2/<name>/blobs/uploads/`                                    | `202`       | `404`
| end-4b  | `POST`         | `/v2/<name>/blobs/uploads/?digest=<digest>`                    | `201`/`202` | `404`/`400`
| end-5   | `PATCH`        | `/v2/<name>/blobs/uploads/<reference>`                         | `202`       | `404`/`416`
| end-6   | `PUT`          | `/v2/<name>/blobs/uploads/<reference>?digest=<digest>`         | `201`       | `404`/`400`
| end-7   | `PUT`          | `/v2/<name>/manifests/<reference>`                             | `201`       | `404`
| end-8a  | `GET`          | `/v2/<name>/tags/list`                                         | `200`       | `404`
| end-8b  | `GET`          | `/v2/<name>/tags/list?n=<integer>&last=<tagname>`              | `200`       | `404`
| end-9   | `DELETE`       | `/v2/<name>/manifests/<reference>`                             | `202`       | `404`/`400`
|         |                |                                                                |             |   /`405`
| end-10  | `DELETE`       | `/v2/<name>/blobs/<digest>`                                    | `202`       | `404`/`405`
| end-11  | `POST`         | `/v2/<name>/blobs/uploads/?mount=<digest>&from=<other_name>`   | `201`       | `404`
| end-12a | `GET`          | `/v2/<name>/referrers/<digest>`                                | `200`       | `404`/`400`
| end-12b | `GET`          | `/v2/<name>/referrers/<digest>?artifactType=<artifactType>`    | `200`       | `404`/`400`
| end-13  | `GET`          | `/v2/<name>/blobs/uploads/<reference>`                         | `204`       | `404`
|.
*/
type Registry interface {
	pkg.Artifact

	// end-1.
	Base() error

	// end-2 HEAD / GET
	HeadBlob(
		ctx2 context.Context,
		artInfo pkg.RegistryInfo,
	) (
		responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64,
		readCloser io.ReadCloser, redirectURL string, errs []error,
	)
	GetBlob(
		ctx2 context.Context,
		artInfo pkg.RegistryInfo,
	) (
		responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64,
		readCloser io.ReadCloser, redirectURL string, errs []error,
	)

	// end-3 HEAD
	ManifestExist(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		acceptHeaders []string,
		ifNoneMatchHeader []string,
	) (
		responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifest manifest.Manifest,
		Errors []error,
	)
	// end-3 GET
	PullManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		acceptHeaders []string,
		ifNoneMatchHeader []string,
	) (
		responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifest manifest.Manifest,
		Errors []error,
	)

	// end-4a.
	PushBlobMonolith(ctx context.Context, artInfo pkg.RegistryInfo, size int64, blob io.Reader) error
	InitBlobUpload(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		fromRepo, mountDigest string,
	) (*commons.ResponseHeaders, []error)
	// end-4b
	PushBlobMonolithWithDigest(ctx context.Context, artInfo pkg.RegistryInfo, size int64, blob io.Reader) error

	// end-5
	PushBlobChunk(
		ctx *Context,
		artInfo pkg.RegistryInfo,
		contentType string,
		contentRange string,
		contentLength string,
		body io.ReadCloser,
		contentLengthFromRequest int64,
	) (*commons.ResponseHeaders, []error)

	// end-6
	PushBlob(
		ctx2 context.Context,
		artInfo pkg.RegistryInfo,
		body io.ReadCloser,
		contentLength int64,
		stateToken string,
	) (*commons.ResponseHeaders, []error)

	// end-7
	PutManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		mediaType string,
		body io.ReadCloser,
		length int64,
	) (*commons.ResponseHeaders, []error)

	// end-8a
	ListTags(
		c context.Context,
		lastEntry string,
		maxEntries int,
		origURL string,
		artInfo pkg.RegistryInfo,
	) (*commons.ResponseHeaders, []string, error)
	// end-8b
	ListFilteredTags(
		ctx context.Context,
		n int,
		last, repository string,
		artInfo pkg.RegistryInfo,
	) (tags []string, err error)

	// end-9
	DeleteManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
	) (errs []error, responseHeaders *commons.ResponseHeaders)
	// the "reference" can be "tag" or "digest", the function needs to handle both

	// end-10.
	DeleteBlob(ctx *Context, artInfo pkg.RegistryInfo) (responseHeaders *commons.ResponseHeaders, errs []error)

	// end-11.
	MountBlob(ctx context.Context, artInfo pkg.RegistryInfo, srcRepository, dstRepository string) (err error)

	// end-12a/12b
	ListReferrers(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		artifactType string,
	) (index *v1.Index, responseHeaders *commons.ResponseHeaders, err error)

	// end-13.
	GetBlobUploadStatus(ctx *Context, artInfo pkg.RegistryInfo, stateToken string) (*commons.ResponseHeaders, []error)

	// Catalog GET.
	GetCatalog() (repositories []string, err error)
	// Tag DELETE.
	DeleteTag(repository, tag string, artInfo pkg.RegistryInfo) error
	// Blob chunk PULL
	PullBlobChunk(
		repository, digest string,
		blobSize, start, end int64,
		artInfo pkg.RegistryInfo,
	) (size int64, blob io.ReadCloser, err error)
	// Mount check
	CanBeMount() (mount bool, repository string, err error)
}
