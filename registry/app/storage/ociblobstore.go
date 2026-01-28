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

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

type ociBlobStore struct {
	repoKey string
	driver  driver.StorageDriver
	// only to be used where context can't come through method args
	ctx                    context.Context
	deleteEnabled          bool
	resumableDigestEnabled bool
	pathFn                 func(pathPrefix string, dgst digest.Digest) (string, error)
	redirect               bool // allows disabling RedirectURL redirects
	rootParentRef          string
}

var _ OciBlobStore = &ociBlobStore{}

func (bs *ociBlobStore) Path() string {
	return bs.rootParentRef
}

// Create begins a blob write session, returning a handle.
func (bs *ociBlobStore) Create(ctx context.Context) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*ociBlobStore).Create")
	uuid := uuid.NewString()

	path, err := pathFor(
		uploadDataPathSpec{
			path:     bs.rootParentRef,
			repoName: bs.repoKey,
			id:       uuid,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, uuid, path, false)
}

func (bs *ociBlobStore) Resume(ctx context.Context, id string) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*ociBlobStore).Resume")

	path, err := pathFor(
		uploadDataPathSpec{
			path:     bs.rootParentRef,
			repoName: bs.repoKey,
			id:       id,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, id, path, true)
}

func (bs *ociBlobStore) Delete(_ context.Context, _ string, _ digest.Digest) error {
	return ErrUnsupported
}

func (bs *ociBlobStore) ServeBlobInternal(
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
	path, err := bs.pathFn(pathPrefix, desc.Digest)
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

func (bs *ociBlobStore) GetBlobInternal(
	ctx context.Context,
	pathPrefix string,
	dgst digest.Digest,
) (*FileReader, int64, error) {
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, 0, err
	}
	size := desc.Size
	path, err := bs.pathFn(pathPrefix, desc.Digest)
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

func (bs *ociBlobStore) Get(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) ([]byte, error) {
	canonical, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, err
	}

	bp, err := bs.pathFn(pathPrefix, canonical.Digest)
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

func (bs *ociBlobStore) Open(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (io.ReadSeekCloser, error) {
	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		return nil, err
	}

	path, err := bs.pathFn(pathPrefix, desc.Digest)
	if err != nil {
		return nil, err
	}

	return NewFileReader(ctx, bs.driver, path, desc.Size)
}

// Put stores the content p in the blob store, calculating the digest.
// If thebcontent is already present, only the digest will be returned.
// This shouldbonly be used for small objects, such as manifests.
// This implemented as a convenience for other Put implementations.
func (bs *ociBlobStore) Put(
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

	bp, err := bs.pathFn(pathPrefix, dgst)
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
func (bs *ociBlobStore) Stat(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (manifest.Descriptor, error) {
	path, err := pathFor(
		blobDataPathSpec{
			digest: dgst,
			path:   pathPrefix,
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

// newBlobUpload allocates a new upload controller with the given state.
func (bs *ociBlobStore) newBlobUpload(
	ctx context.Context, uuid,
	path string, a bool,
) (BlobWriter, error) {
	fw, err := bs.driver.Writer(ctx, path, a)
	if err != nil {
		return nil, err
	}

	bw := &blobWriter{
		ctx:                    ctx,
		blobStore:              bs,
		id:                     uuid,
		digester:               digest.Canonical.Digester(),
		fileWriter:             fw,
		driver:                 bs.driver,
		path:                   path,
		resumableDigestEnabled: bs.resumableDigestEnabled,
	}

	return bw, nil
}
