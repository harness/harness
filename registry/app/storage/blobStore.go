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

package storage

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/types"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

type genericBlobStore struct {
	driver        driver.StorageDriver
	rootParentRef string
	redirect      bool
}

var _ GenericBlobStore = &genericBlobStore{}

func (bs *genericBlobStore) GetV2NoRedirect(
	ctx context.Context,
	rootIdentifier string,
	sha256 string,
	fileSize int64,
) (*FileReader, error) {
	log.Ctx(ctx).Debug().Msg("(*genericBlobStore).GetV2")

	path, err := pathFor(
		genericDataPathSpec{
			rootIdentifier: rootIdentifier,
			sha256:         sha256,
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

func (bs *genericBlobStore) GetGeneric(
	ctx context.Context,
	size int64,
	filename string,
	rootIdentifier string,
	sha256 string,
) (*FileReader, string, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Get")

	path, err := pathFor(
		genericDataPathSpec{
			rootIdentifier: rootIdentifier,
			sha256:         sha256,
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
func (bs *genericBlobStore) CreateGeneric(ctx context.Context, rootIdentifier string) (BlobWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Create")

	id := uuid.NewString()
	path, err := pathFor(
		genericUploadDataPathSpec{
			rootIdentifier: rootIdentifier,
			id:             id,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, id, path, rootIdentifier, false)
}

func (bs *genericBlobStore) newBlobUpload(ctx context.Context, id, path, rootIdentifier string, a bool) (
	BlobWriter,
	error,
) {
	fw, err := bs.driver.Writer(ctx, path, a)
	if err != nil {
		return nil, err
	}

	bw := &blobWriter{
		ctx:                    ctx,
		genericBlobStore:       bs,
		id:                     id,
		digester:               digest.Canonical.Digester(),
		fileWriter:             fw,
		driver:                 bs.driver,
		path:                   path,
		resumableDigestEnabled: true,
		rootIdentifier:         rootIdentifier,
	}

	return bw, nil
}

// Write takes a file writer and a multipart form file or file reader,
// streams the file to the writer, and calculates hashes.
func (bs *genericBlobStore) Write(
	ctx context.Context, w BlobWriter, file multipart.File,
	fileReader io.Reader,
) (types.FileInfo, error) {
	// Create new hash.Hash instances for SHA256 and SHA512
	sha1Hasher := sha1.New()
	sha256Hasher := sha256.New()
	sha512Hasher := sha512.New()
	md5Hasher := md5.New()

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

func (bs *genericBlobStore) move(
	ctx context.Context,
	rootIdentifier string,
	id string,
	sha256 string,
) error {
	log.Ctx(ctx).Debug().Msg("(*genericBlobStore).Move")
	srcPath, err := pathFor(
		genericUploadDataPathSpec{
			rootIdentifier: rootIdentifier,
			id:             id,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create srcPath for root: %s, id: %s, digest: %s, %w", rootIdentifier, id, sha256,
			err)
	}
	dstPath, err := pathFor(
		genericDataPathSpec{
			rootIdentifier: rootIdentifier,
			sha256:         sha256,
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

func (bs *genericBlobStore) Delete(ctx context.Context, filePath string) error {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Delete")
	err := bs.driver.Delete(ctx, filePath)
	if err != nil {
		return err
	}
	return nil
}

func (bs *genericBlobStore) StatByDigest(ctx context.Context, rootIdentifier, sha256 string) (int64, error) {
	log.Ctx(ctx).Debug().Msg("(*genericBlobStore).StatByDigest")

	path, err := pathFor(
		genericDataPathSpec{
			rootIdentifier: rootIdentifier,
			sha256:         sha256,
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

func (bs *genericBlobStore) BucketKey() types.BucketKey {
	return DefaultBucketKey
}
