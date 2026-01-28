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

package storage

import (
	"context"
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/types"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// blobStore
/*
Important notes:
 1.This relies on digest.Digest as Storage relies only on sha256 and cannot rely on types.Digest. Any caller should
   do the conversion before reaching here.
 2.All Path for S3 should remain here.
*/
type blobStore struct {
	DriverMeta DriverResult
	driver     driver.StorageDriver
	// only to be used where context can't come through method args
	ctx                    context.Context
	resumableDigestEnabled bool
	redirect               bool
	deleteEnabled          bool
	// To be cleaned up
	rootParentRef    string
	repoKey          string
	multipartEnabled bool
}

var _ OciBlobStore = &blobStore{}
var _ GenericBlobStore = &blobStore{}

func (bs *blobStore) GetV2NoRedirect(
	ctx context.Context,
	_ string,
	sha256 string,
	fileSize int64,
) (*FileReader, error) {
	log.Ctx(ctx).Debug().Msg("(*globalBlobStore).GetV2")

	path, err := pathFor(
		globalBlobPathSpec{
			digest: digest.Digest(sha256),
		},
	)

	if err != nil {
		return nil, err
	}

	br, err := NewFileReader(ctx, bs.driver, path, fileSize)
	if err != nil {
		return nil, err
	}
	return br, nil
}

func (bs *blobStore) GetGeneric(
	ctx context.Context,
	size int64,
	filename string,
	_ string,
	sha256 string,
) (*FileReader, string, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*globalBlobStore).Get")

	path, err := pathFor(
		globalBlobPathSpec{
			digest: digest.Digest(sha256),
		},
	)

	if err != nil {
		return nil, "", err
	}

	if bs.redirect {
		redirectURL, err := bs.driver.RedirectURL(ctx, http.MethodGet, path, filename)
		if err != nil {
			return nil, "", err
		}
		if redirectURL != "" {
			// Redirect to storage URL.
			// http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return nil, redirectURL, nil
		}
		// Fallback to serving the content directly.
	}
	br, err := NewFileReader(ctx, bs.driver, path, size)
	if err != nil {
		return nil, "", err
	}
	return br, "", nil
}

// Create begins a blob write session, returning a handle.
func (bs *blobStore) CreateGeneric(ctx context.Context, rootIdentifier string) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*globalBlobStore).Create")

	id := uuid.NewString()
	path, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, id, path, rootIdentifier, false)
}

func (bs *blobStore) newBlobUpload(ctx context.Context, id, path, rootIdentifier string, appendMode bool) (
	BlobWriter,
	error,
) {
	fw, err := bs.driver.Writer(ctx, path, appendMode)
	if err != nil {
		return nil, err
	}

	bw := &blobWriter{
		ctx:                    ctx,
		globalBlobStore:        bs,
		id:                     id,
		digester:               digest.Canonical.Digester(),
		fileWriter:             fw,
		driver:                 bs.driver,
		path:                   path,
		resumableDigestEnabled: true,
		rootIdentifier:         rootIdentifier,
		isMultiPart:            bs.multipartEnabled,
	}

	return bw, nil
}

// Write takes a file writer and a multipart form file or file reader,
// streams the file to the writer, and calculates hashes.
func (bs *blobStore) Write(
	ctx context.Context, w BlobWriter, file multipart.File,
	fileReader io.Reader,
) (types.FileInfo, error) {
	// Create new hash.Hash instances for SHA256 and SHA512
	sha1Hasher := sha1.New() //nolint:gosec
	sha256Hasher := sha256.New()
	sha512Hasher := sha512.New()
	md5Hasher := md5.New() //nolint:gosec

	// Create a MultiWriter to write to both hashers simultaneously
	mw := io.MultiWriter(sha1Hasher, sha256Hasher, sha512Hasher, md5Hasher, w)
	// Copy the data from S3 object stream to the MultiWriter
	var err error
	var totalBytesWritten int64
	if fileReader != nil {
		totalBytesWritten, err = io.Copy(mw, fileReader)
	} else {
		totalBytesWritten, err = io.Copy(mw, file)
	}
	if err != nil {
		return types.FileInfo{}, fmt.Errorf("failed to copy file to s3: %w", err)
	}

	return types.FileInfo{
		Sha1:   fmt.Sprintf("%x", sha1Hasher.Sum(nil)),
		Sha256: fmt.Sprintf("%x", sha256Hasher.Sum(nil)),
		Sha512: fmt.Sprintf("%x", sha512Hasher.Sum(nil)),
		MD5:    fmt.Sprintf("%x", md5Hasher.Sum(nil)),
		Size:   totalBytesWritten,
	}, nil
}

func (bs *blobStore) move(
	ctx context.Context,
	rootIdentifier string,
	id string,
	sha256 string,
) error {
	log.Ctx(ctx).Debug().Msg("(*globalBlobStore).Move")
	srcPath, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create srcPath for root: %s, id: %s, digest: %s, %w", rootIdentifier, id, sha256,
			err)
	}
	dstPath, err := pathFor(
		globalBlobPathSpec{
			digest: digest.Digest(sha256),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create dstPath for root: %s, id: %s, digest: %s, %w", rootIdentifier, id, sha256,
			err)
	}
	err = bs.driver.Move(ctx, srcPath, dstPath)
	if err != nil {
		return err
	}
	return nil
}

func (bs *blobStore) StatByDigest(ctx context.Context, rootIdentifier, sha256 string) (int64, error) {
	log.Ctx(ctx).Debug().Msg("(*globalBlobStore).StatByDigest")

	path, err := pathFor(
		globalBlobPathSpec{
			digest: digest.Digest(sha256),
		},
	)

	if err != nil {
		return 0, err
	}

	fileInfo, err := bs.driver.Stat(ctx, path)
	if err != nil {
		return -1, err
	}
	return fileInfo.Size(), nil
}

func (bs *blobStore) GetDriverDetails() DriverResult {
	return bs.DriverMeta
}

func (bs *blobStore) Path() string {
	return ""
}

// Create begins a blob write session, returning a handle.
func (bs *blobStore) Create(ctx context.Context) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*ociBlobStore).Create")
	uuid := uuid.NewString()

	path, err := pathFor(
		globalUploadDataPathSpec{
			id: uuid,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, uuid, path, bs.rootParentRef, false)
}

func (bs *blobStore) Resume(ctx context.Context, id string) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*ociBlobStore).Resume")

	path, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, id, path, bs.rootParentRef, true)
}

func (bs *blobStore) ServeBlobInternal(
	ctx context.Context,
	pathPrefix string,
	dgst digest.Digest,
	headers map[string]string,
	method string,
) (*FileReader, string, int64, error) {
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, "", 0, err
	}
	if desc.MediaType != "" {
		// Set the repository local content type.
		headers[HeaderContentType] = desc.MediaType
	}
	size := desc.Size
	path, err := bs.globalPathFn(desc.Digest)
	if err != nil {
		return nil, "", size, err
	}

	if bs.redirect {
		redirectURL, err := bs.driver.RedirectURL(ctx, method, path, "")
		if err != nil {
			return nil, "", size, err
		}
		if redirectURL != "" {
			// Redirect to storage URL.
			// http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return nil, redirectURL, size, nil
		}
		// Fallback to serving the content directly.
	}

	br, err := NewFileReader(ctx, bs.driver, path, desc.Size)
	if err != nil {
		if br != nil {
			br.Close()
		}
		return nil, "", size, err
	}

	headers[HeaderEtag] = fmt.Sprintf(`"%s"`, desc.Digest)
	// If-None-Match handled by ServeContent
	headers[HeaderCacheControl] = fmt.Sprintf(
		"max-age=%.f",
		blobCacheControlMaxAge.Seconds(),
	)

	if headers[HeaderDockerContentDigest] == "" {
		headers[HeaderDockerContentDigest] = desc.Digest.String()
	}

	if headers[HeaderContentType] == "" {
		// Set the content type if not already set.
		headers[HeaderContentType] = desc.MediaType
	}

	if headers[HeaderContentLength] == "" {
		// Set the content length if not already set.
		headers[HeaderContentLength] = fmt.Sprint(desc.Size)
	}

	return br, "", size, err
}

func (bs *blobStore) GetBlobInternal(
	ctx context.Context,
	pathPrefix string,
	dgst digest.Digest,
) (*FileReader, int64, error) {
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, 0, err
	}
	size := desc.Size
	path, err := bs.globalPathFn(desc.Digest)
	if err != nil {
		return nil, size, err
	}

	br, err := NewFileReader(ctx, bs.driver, path, desc.Size)
	if err != nil {
		if br != nil {
			br.Close()
		}
		return nil, size, err
	}
	return br, size, err
}

func (bs *blobStore) Get(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) ([]byte, error) {
	canonical, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, err
	}

	bp, err := bs.globalPathFn(canonical.Digest)
	if err != nil {
		return nil, err
	}

	p, err := getContent(ctx, bs.driver, bp)
	if err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			return nil, ErrBlobUnknown
		}
		return nil, err
	}

	return p, nil
}

func (bs *blobStore) Open(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (io.ReadSeekCloser, error) {
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, err
	}

	path, err := bs.globalPathFn(desc.Digest)
	if err != nil {
		return nil, err
	}

	return NewFileReader(ctx, bs.driver, path, desc.Size)
}

// Put stores the content p in the blob store, calculating the digest.
// If thebcontent is already present, only the digest will be returned.
// This shouldbonly be used for small objects, such as manifests.
// This implemented as a convenience for other Put implementations.
func (bs *blobStore) Put(
	ctx context.Context, pathPrefix string,
	p []byte,
) (manifest.Descriptor, error) {
	dgst := digest.FromBytes(p)
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err == nil {
		// content already present
		return desc, nil
	} else if !errors.Is(err, ErrBlobUnknown) {
		dcontext.GetLogger(
			ctx, log.Error(),
		).Msgf(
			"ociBlobStore: error stating content (%v): %v", dgst, err,
		)
		// real error, return it
		return manifest.Descriptor{}, err
	}

	bp, err := bs.globalPathFn(dgst)
	if err != nil {
		return manifest.Descriptor{}, err
	}

	return manifest.Descriptor{
		Size: int64(len(p)),

		MediaType: "application/octet-stream",
		Digest:    dgst,
	}, bs.driver.PutContent(ctx, bp, p)
}

// Stat returns the descriptor for the blob
// in the main blob store. If this method returns successfully, there is
// strong guarantee that the blob exists and is available.
func (bs *blobStore) Stat(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (manifest.Descriptor, error) {
	path, err := pathFor(
		globalBlobPathSpec{
			digest: dgst,
		},
	)
	if err != nil {
		return manifest.Descriptor{}, err
	}

	fi, err := bs.driver.Stat(ctx, path)
	if err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			return manifest.Descriptor{}, ErrBlobUnknown
		}
		return manifest.Descriptor{}, err
	}

	if fi.IsDir() {
		dcontext.GetLogger(
			ctx, log.Warn(),
		).Msgf("blob path should not be a directory: %q", path)
		return manifest.Descriptor{}, ErrBlobUnknown
	}

	return manifest.Descriptor{
		Size: fi.Size(),

		MediaType: "application/octet-stream",
		Digest:    dgst,
	}, nil
}

func (bs *blobStore) globalPathFn(dgst digest.Digest) (string, error) {
	bp, err := pathFor(
		globalBlobPathSpec{
			digest: dgst,
		},
	)
	if err != nil {
		return "", err
	}

	return bp, nil
}
