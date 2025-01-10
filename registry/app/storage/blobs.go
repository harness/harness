// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
)

var (
	// ErrBlobExists returned when blob already exists.
	ErrBlobExists = errors.New("blob exists")

	// ErrBlobDigestUnsupported when blob digest is an unsupported version.
	ErrBlobDigestUnsupported = errors.New("unsupported blob digest")

	// ErrBlobUnknown when blob is not found.
	ErrBlobUnknown = errors.New("unknown blob")

	// ErrBlobUploadUnknown returned when upload is not found.
	ErrBlobUploadUnknown = errors.New("blob upload unknown")

	// ErrBlobInvalidLength returned when the blob has an expected length on
	// commit, meaning mismatched with the descriptor or an invalid value.
	ErrBlobInvalidLength = errors.New("blob invalid length")
)

// BlobInvalidDigestError returned when digest check fails.
type BlobInvalidDigestError struct {
	Digest digest.Digest
	Reason error
}

func (err BlobInvalidDigestError) Error() string {
	return fmt.Sprintf(
		"invalid digest for referenced layer: %v, %v",
		err.Digest, err.Reason,
	)
}

// BlobMountedError returned when a blob is mounted from another repository
// instead of initiating an upload session.
type BlobMountedError struct {
	From       reference.Canonical
	Descriptor manifest.Descriptor
}

func (err BlobMountedError) Error() string {
	return fmt.Sprintf(
		"blob mounted from: %v to: %v",
		err.From, err.Descriptor,
	)
}

// BlobWriter provides a handle for inserting data into a blob store.
// Instances should be obtained from BlobWriteService.Writer and
// BlobWriteService.Resume. If supported by the store, a writer can be
// recovered with the id.
type BlobWriter interface {
	io.WriteCloser

	// Size returns the number of bytes written to this blob.
	Size() int64

	// ID returns the identifier for this writer. The ID can be used with the
	// Blob service to later resume the write.
	ID() string

	// Commit completes the blob writer process. The content is verified
	// against the provided provisional descriptor, which may result in an
	// error. Depending on the implementation, written data may be validated
	// against the provisional descriptor fields. If MediaType is not present,
	// the implementation may reject the commit or assign "application/octet-
	// stream" to the blob. The returned descriptor may have a different
	// digest depending on the blob store, referred to as the canonical
	// descriptor.
	Commit(ctx context.Context, pathPrefix string, provisional manifest.Descriptor) (
		canonical manifest.Descriptor, err error,
	)

	// Cancel ends the blob write without storing any data and frees any
	// associated resources. Any data written thus far will be lost. Cancel
	// implementations should allow multiple calls even after a commit that
	// result in a no-op. This allows use of Cancel in a defer statement,
	// increasing the assurance that it is correctly called.
	Cancel(ctx context.Context) error
}

// OciBlobStore represent the entire suite of blob related operations. Such an
// implementation can access, read, write, delete and serve blobs.
type OciBlobStore interface {

	// ServeBlobInternal attempts to serve the blob, identified by dgst, via http. The
	// service may decide to redirect the client elsewhere or serve the data
	// directly.
	//
	// This handler only issues successful responses, such as 2xx or 3xx,
	// meaning it serves data or issues a redirect. If the blob is not
	// available, an error will be returned and the caller may still issue a
	// response.
	//
	// The implementation may serve the same blob from a different digest
	// domain. The appropriate headers will be set for the blob, unless they
	// have already been set by the caller.
	ServeBlobInternal(
		ctx context.Context,
		pathPrefix string,
		dgst digest.Digest,
		headers map[string]string,
		method string,
	) (*FileReader, string, int64, error)

	Delete(ctx context.Context, pathPrefix string, dgst digest.Digest) error

	// Stat provides metadata about a blob identified by the digest. If the
	// blob is unknown to the describer, ErrBlobUnknown will be returned.
	Stat(ctx context.Context, pathPrefix string, dgst digest.Digest) (manifest.Descriptor, error)

	// Get returns the entire blob identified by digest along with the descriptor.
	Get(ctx context.Context, pathPrefix string, dgst digest.Digest) ([]byte, error)

	// Open provides an [io.ReadSeekCloser] to the blob identified by the provided
	// descriptor. If the blob is not known to the service, an error is returned.
	Open(ctx context.Context, pathPrefix string, dgst digest.Digest) (io.ReadSeekCloser, error)

	// Put inserts the content p into the blob service, returning a descriptor
	// or an error.
	Put(ctx context.Context, pathPrefix string, p []byte) (manifest.Descriptor, error)

	// Create allocates a new blob writer to add a blob to this service. The
	// returned handle can be written to and later resumed using an opaque
	// identifier. With this approach, one can Close and Resume a BlobWriter
	// multiple times until the BlobWriter is committed or cancelled.
	Create(ctx context.Context) (BlobWriter, error)

	// Resume attempts to resume a write to a blob, identified by an id.
	Resume(ctx context.Context, id string) (BlobWriter, error)

	Path() string
}

// GenericBlobStore represent the entire suite of Generic blob related operations. Such an
// implementation can access, read, write, delete and serve blobs.
type GenericBlobStore interface {

	// Create allocates a new blob writer to add a blob to this service. The
	// returned handle can be written to and later resumed using an opaque
	// identifier. With this approach, one can Close and Resume a BlobWriter
	// multiple times until the BlobWriter is committed or cancelled.
	Create(ctx context.Context, filePath string) (driver.FileWriter, error)

	Write(ctx context.Context, w driver.FileWriter, file multipart.File) (pkg.FileInfo, error)
	Move(ctx context.Context, srcPath string, dstPath string) error
	Delete(ctx context.Context, filePath string) error
}
