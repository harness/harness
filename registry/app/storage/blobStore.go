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

	"github.com/rs/zerolog/log"
)

type genericBlobStore struct {
	repoKey       string
	driver        driver.StorageDriver
	rootParentRef string
	redirect      bool
}

func (bs *genericBlobStore) Info() string {
	return bs.rootParentRef + " " + bs.repoKey
}

func (bs *genericBlobStore) Get(ctx context.Context, filePath string, size int64) (*FileReader, string, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Get")

	if bs.redirect {
		redirectURL, err := bs.driver.RedirectURL(ctx, http.MethodGet, filePath)
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
	br, err := NewFileReader(ctx, bs.driver, filePath, size)
	if err != nil {
		return nil, "", err
	}
	return br, "", nil
}

var _ GenericBlobStore = &genericBlobStore{}

// Create begins a blob write session, returning a handle.
func (bs *genericBlobStore) Create(ctx context.Context, filePath string) (driver.FileWriter, error) {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Create")

	path, err := pathFor(
		uploadFilePathSpec{
			path: filePath,
		},
	)
	if err != nil {
		return nil, err
	}

	return bs.newBlobUpload(ctx, path, false)
}

func (bs *genericBlobStore) newBlobUpload(
	ctx context.Context,
	path string, a bool,
) (driver.FileWriter, error) {
	fw, err := bs.driver.Writer(ctx, path, a)
	if err != nil {
		return nil, err
	}
	return fw, nil
}

// Write takes a file writer and a multipart form file or file reader,
// streams the file to the writer, and calculates hashes.
func (bs *genericBlobStore) Write(
	ctx context.Context, w driver.FileWriter, file multipart.File,
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

	err = w.Commit(ctx)
	if err != nil {
		return types.FileInfo{}, err
	}

	return types.FileInfo{
		Sha1:   fmt.Sprintf("%x", sha1Hasher.Sum(nil)),
		Sha256: fmt.Sprintf("%x", sha256Hasher.Sum(nil)),
		Sha512: fmt.Sprintf("%x", sha512Hasher.Sum(nil)),
		MD5:    fmt.Sprintf("%x", md5Hasher.Sum(nil)),
		Size:   totalBytesWritten,
	}, nil
}

func (bs *genericBlobStore) Move(ctx context.Context, srcPath string, dstPath string) error {
	dcontext.GetLogger(ctx, log.Ctx(ctx).Debug()).Msg("(*genericBlobStore).Move")
	err := bs.driver.Move(ctx, srcPath, dstPath)
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
