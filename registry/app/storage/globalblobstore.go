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

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/types"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// blobStore implements both OciBlobStore and GenericBlobStore for global/non-default buckets.
/*
Important notes:
 1. This relies on digest.Digest as Storage relies only on sha256 and cannot rely on types.Digest.
    Any caller should do the conversion before reaching here.
 2. All Path for S3 should remain here.
*/
type globalBlobStore struct {
	bucketKey              types.BucketKey
	driver                 driver.StorageDriver
	ctx                    context.Context
	resumableDigestEnabled bool
	redirect               bool
	deleteEnabled          bool
	multipartEnabled       bool
}

var _ OciBlobStore = &globalBlobStore{}
var _ GenericBlobStore = &globalBlobStore{}

func (bs *globalBlobStore) GetV2NoRedirect(
	ctx context.Context,
	_ string,
	sha256 string,
	fileSize int64,
) (*FileReader, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetV2NoRedirect").
		Int64("file_size", fileSize).
		Msg("starting blob retrieval without redirect")

	path, err := GlobalPathFn(digest.Digest(sha256))
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetV2NoRedirect").
			Err(err).
			Msg("failed to get global path")
		return nil, err
	}

	br, err := NewFileReader(ctx, bs.driver, path, fileSize)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetV2NoRedirect").
			Str("path", path).
			Err(err).
			Msg("failed to create file reader")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetV2NoRedirect").
		Str("path", path).
		Msg("blob retrieval successful")
	return br, nil
}

func (bs *globalBlobStore) GetGeneric(
	ctx context.Context,
	size int64,
	filename string,
	_ string,
	sha256 string,
) (*FileReader, string, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetGeneric").
		Int64("size", size).
		Str("filename", filename).
		Bool("redirect_enabled", bs.redirect).
		Msg("starting generic blob retrieval")

	path, err := GlobalPathFn(digest.NewDigestFromEncoded(digest.SHA256, sha256))
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetGeneric").
			Err(err).
			Msg("failed to get global path")
		return nil, "", err
	}

	if bs.redirect {
		redirectURL, err := bs.driver.RedirectURL(ctx, http.MethodGet, path, filename)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.GetGeneric").
				Str("path", path).
				Err(err).
				Msg("failed to get redirect URL")
			return nil, "", err
		}
		if redirectURL != "" {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.GetGeneric").
				Str("path", path).
				Msg("returning redirect URL")
			return nil, redirectURL, nil
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetGeneric").
			Str("path", path).
			Msg("redirect URL empty, falling back to direct content")
	}

	br, err := NewFileReader(ctx, bs.driver, path, size)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetGeneric").
			Str("path", path).
			Err(err).
			Msg("failed to create file reader")
		return nil, "", err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetGeneric").
		Str("path", path).
		Msg("generic blob retrieval successful")
	return br, "", nil
}

// Create begins a blob write session, returning a handle.
func (bs *globalBlobStore) CreateGeneric(ctx context.Context, rootIdentifier string) (BlobWriter, error) {
	id := uuid.NewString()

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.CreateGeneric").
		Str("root_identifier", rootIdentifier).
		Str("upload_id", id).
		Msg("creating generic blob upload session")

	path, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.CreateGeneric").
			Str("upload_id", id).
			Err(err).
			Msg("failed to create upload path")
		return nil, err
	}

	bw, err := bs.newBlobUpload(ctx, id, path, false)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.CreateGeneric").
			Str("upload_id", id).
			Str("path", path).
			Err(err).
			Msg("failed to create blob upload")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.CreateGeneric").
		Str("upload_id", id).
		Str("path", path).
		Msg("generic blob upload session created")
	return bw, nil
}

func (bs *globalBlobStore) newBlobUpload(ctx context.Context, id, path string, appendMode bool) (BlobWriter, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.newBlobUpload").
		Str("upload_id", id).
		Str("path", path).
		Msg("initializing blob upload writer")

	fw, err := bs.driver.Writer(ctx, path, appendMode)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.newBlobUpload").
			Str("upload_id", id).
			Str("path", path).
			Err(err).
			Msg("failed to create driver writer")
		return nil, err
	}

	bw := &globalBlobWriter{
		ctx:                    ctx,
		globalBlobStore:        bs,
		id:                     id,
		digester:               digest.Canonical.Digester(),
		fileWriter:             fw,
		driver:                 bs.driver,
		path:                   path,
		resumableDigestEnabled: true,
		isMultiPart:            bs.multipartEnabled,
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.newBlobUpload").
		Str("upload_id", id).
		Str("path", path).
		Msg("blob upload writer initialized")
	return bw, nil
}

// Write takes a file writer and a multipart form file or file reader,
// streams the file to the writer, and calculates hashes.
func (bs *globalBlobStore) Write(
	ctx context.Context, w BlobWriter, file multipart.File,
	fileReader io.Reader,
) (types.FileInfo, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Write").
		Bool("has_file_reader", fileReader != nil).
		Bool("has_file", file != nil).
		Msg("starting blob write with hash calculation")

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
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Write").
			Err(err).
			Msg("failed to copy file to storage")
		return types.FileInfo{}, fmt.Errorf("failed to copy file: %w", err)
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Write").
		Int64("bytes_written", totalBytesWritten).
		Msg("blob write completed successfully")

	return types.FileInfo{
		Sha1:   fmt.Sprintf("%x", sha1Hasher.Sum(nil)),
		Sha256: fmt.Sprintf("%x", sha256Hasher.Sum(nil)),
		Sha512: fmt.Sprintf("%x", sha512Hasher.Sum(nil)),
		MD5:    fmt.Sprintf("%x", md5Hasher.Sum(nil)),
		Size:   totalBytesWritten,
	}, nil
}

func (bs *globalBlobStore) move(ctx context.Context, id string, sha256 string) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.move").
		Str("upload_id", id).
		Msg("starting blob move to permanent location")

	srcPath, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.move").
			Str("upload_id", id).
			Err(err).
			Msg("failed to create source path")
		return fmt.Errorf("failed to create srcPath id: %s, digest: %s, %w", id, sha256, err)
	}

	dstPath, err := GlobalPathFn(digest.NewDigestFromEncoded(digest.SHA256, sha256))
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.move").
			Str("upload_id", id).
			Str("src_path", srcPath).
			Err(err).
			Msg("failed to create destination path")
		return fmt.Errorf("failed to create dstPath id: %s, digest: %s, %w", id, sha256, err)
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.move").
		Str("upload_id", id).
		Str("src_path", srcPath).
		Str("dst_path", dstPath).
		Msg("moving blob from upload to permanent location")

	err = bs.driver.Move(ctx, srcPath, dstPath)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.move").
			Str("upload_id", id).
			Str("src_path", srcPath).
			Str("dst_path", dstPath).
			Err(err).
			Msg("failed to move blob")
		return err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.move").
		Str("upload_id", id).
		Str("dst_path", dstPath).
		Msg("blob moved successfully")
	return nil
}

func (bs *globalBlobStore) StatByDigest(ctx context.Context, rootIdentifier, sha256 string) (int64, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.StatByDigest").
		Str("root_identifier", rootIdentifier).
		Msg("starting stat by digest")

	path, err := GlobalPathFn(digest.NewDigestFromEncoded(digest.SHA256, sha256))
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.StatByDigest").
			Err(err).
			Msg("failed to get global path")
		return 0, err
	}

	fileInfo, err := bs.driver.Stat(ctx, path)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.StatByDigest").
			Str("path", path).
			Err(err).
			Msg("failed to stat blob")
		return -1, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.StatByDigest").
		Str("path", path).
		Int64("size", fileInfo.Size()).
		Msg("stat by digest completed")
	return fileInfo.Size(), nil
}

func (bs *globalBlobStore) BucketKey() types.BucketKey {
	log.Ctx(bs.ctx).Debug().
		Str("method", "globalBlobStore.BucketKey").
		Str("bucket_key", string(bs.bucketKey)).
		Msg("returning bucket key")
	return bs.bucketKey
}

func (bs *globalBlobStore) Path() string {
	return ""
}

// Create begins a blob write session, returning a handle.
func (bs *globalBlobStore) Create(ctx context.Context) (BlobWriter, error) {
	id := uuid.NewString()

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Create").
		Str("upload_id", id).
		Msg("creating OCI blob upload session")

	path, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Create").
			Str("upload_id", id).
			Err(err).
			Msg("failed to create upload path")
		return nil, err
	}

	bw, err := bs.newBlobUpload(ctx, id, path, false)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Create").
			Str("upload_id", id).
			Str("path", path).
			Err(err).
			Msg("failed to create blob upload")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Create").
		Str("upload_id", id).
		Str("path", path).
		Msg("OCI blob upload session created")
	return bw, nil
}

func (bs *globalBlobStore) Resume(ctx context.Context, id string) (BlobWriter, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Resume").
		Str("upload_id", id).
		Msg("resuming blob upload session")

	path, err := pathFor(
		globalUploadDataPathSpec{
			id: id,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Resume").
			Str("upload_id", id).
			Err(err).
			Msg("failed to create upload path")
		return nil, err
	}

	bw, err := bs.newBlobUpload(ctx, id, path, true)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Resume").
			Str("upload_id", id).
			Str("path", path).
			Err(err).
			Msg("failed to resume blob upload")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Resume").
		Str("upload_id", id).
		Str("path", path).
		Msg("blob upload session resumed")
	return bw, nil
}

func (bs *globalBlobStore) ServeBlobInternal(
	ctx context.Context,
	pathPrefix string,
	dgst digest.Digest,
	headers map[string]string,
	method string,
) (*FileReader, string, int64, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.ServeBlobInternal").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Str("http_method", method).
		Bool("redirect_enabled", bs.redirect).
		Msg("starting serve blob internal")

	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.ServeBlobInternal").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to stat blob")
		return nil, "", 0, err
	}
	if desc.MediaType != "" {
		// Set the repository local content type.
		headers[HeaderContentType] = desc.MediaType
	}
	size := desc.Size
	path, err := GlobalPathFn(desc.Digest)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.ServeBlobInternal").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return nil, "", size, err
	}

	if bs.redirect {
		redirectURL, err := bs.driver.RedirectURL(ctx, method, path, "")
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.ServeBlobInternal").
				Str("path", path).
				Err(err).
				Msg("failed to get redirect URL")
			return nil, "", size, err
		}
		if redirectURL != "" {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.ServeBlobInternal").
				Str("path", path).
				Int64("size", size).
				Msg("returning redirect URL")
			return nil, redirectURL, size, nil
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.ServeBlobInternal").
			Str("path", path).
			Msg("redirect URL empty, falling back to direct content")
	}

	br, err := NewFileReader(ctx, bs.driver, path, desc.Size)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.ServeBlobInternal").
			Str("path", path).
			Err(err).
			Msg("failed to create file reader")
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

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.ServeBlobInternal").
		Str("path", path).
		Int64("size", size).
		Str("media_type", desc.MediaType).
		Msg("serve blob internal completed")
	return br, "", size, err
}

func (bs *globalBlobStore) GetBlobInternal(
	ctx context.Context,
	pathPrefix string,
	dgst digest.Digest,
) (*FileReader, int64, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetBlobInternal").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Msg("starting get blob internal")

	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetBlobInternal").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to stat blob")
		return nil, 0, err
	}
	size := desc.Size
	path, err := GlobalPathFn(desc.Digest)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetBlobInternal").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return nil, size, err
	}

	br, err := NewFileReader(ctx, bs.driver, path, desc.Size)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.GetBlobInternal").
			Str("path", path).
			Err(err).
			Msg("failed to create file reader")
		if br != nil {
			br.Close()
		}
		return nil, size, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.GetBlobInternal").
		Str("path", path).
		Int64("size", size).
		Msg("get blob internal completed")
	return br, size, err
}

func (bs *globalBlobStore) Get(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) ([]byte, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Get").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Msg("starting blob get")

	canonical, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Get").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to stat blob")
		return nil, err
	}

	bp, err := GlobalPathFn(canonical.Digest)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Get").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return nil, err
	}

	p, err := getContent(ctx, bs.driver, bp)
	if err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.Get").
				Str("path", bp).
				Msg("blob not found")
			return nil, ErrBlobUnknown
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Get").
			Str("path", bp).
			Err(err).
			Msg("failed to get content")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Get").
		Str("path", bp).
		Int("content_length", len(p)).
		Msg("blob get completed")
	return p, nil
}

func (bs *globalBlobStore) Open(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (io.ReadSeekCloser, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Open").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Msg("starting blob open")

	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Open").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to stat blob")
		return nil, err
	}

	path, err := GlobalPathFn(desc.Digest)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Open").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return nil, err
	}

	reader, err := NewFileReader(ctx, bs.driver, path, desc.Size)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Open").
			Str("path", path).
			Err(err).
			Msg("failed to create file reader")
		return nil, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Open").
		Str("path", path).
		Int64("size", desc.Size).
		Msg("blob open completed")
	return reader, nil
}

// Put stores the content p in the blob store, calculating the digest.
// If the content is already present, only the digest will be returned.
// This should only be used for small objects, such as manifests.
// This implemented as a convenience for other Put implementations.
func (bs *globalBlobStore) Put(
	ctx context.Context, pathPrefix string,
	p []byte,
) (manifest.Descriptor, error) {
	dgst := digest.FromBytes(p)

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Put").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Int("content_length", len(p)).
		Msg("starting blob put")

	desc, err := bs.Stat(ctx, pathPrefix, dgst)
	if err == nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Put").
			Str("digest", dgst.String()).
			Msg("content already present, skipping upload")
		return desc, nil
	} else if !errors.Is(err, ErrBlobUnknown) {
		log.Ctx(ctx).Error().
			Str("method", "globalBlobStore.Put").
			Str("digest", dgst.String()).
			Err(err).
			Msg("error stating content")
		return manifest.Descriptor{}, err
	}

	bp, err := GlobalPathFn(dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Put").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return manifest.Descriptor{}, err
	}

	err = bs.driver.PutContent(ctx, bp, p)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Put").
			Str("path", bp).
			Err(err).
			Msg("failed to put content")
		return manifest.Descriptor{}, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Put").
		Str("path", bp).
		Int("content_length", len(p)).
		Msg("blob put completed")

	return manifest.Descriptor{
		Size:      int64(len(p)),
		MediaType: "application/octet-stream",
		Digest:    dgst,
	}, nil
}

// Stat returns the descriptor for the blob
// in the main blob store. If this method returns successfully, there is
// strong guarantee that the blob exists and is available.
func (bs *globalBlobStore) Stat(
	ctx context.Context, pathPrefix string,
	dgst digest.Digest,
) (manifest.Descriptor, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Stat").
		Str("path_prefix", pathPrefix).
		Str("digest", dgst.String()).
		Msg("starting blob stat")

	path, err := GlobalPathFn(dgst)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Stat").
			Str("digest", dgst.String()).
			Err(err).
			Msg("failed to get global path")
		return manifest.Descriptor{}, err
	}

	fi, err := bs.driver.Stat(ctx, path)
	if err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobStore.Stat").
				Str("path", path).
				Msg("blob not found")
			return manifest.Descriptor{}, ErrBlobUnknown
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobStore.Stat").
			Str("path", path).
			Err(err).
			Msg("failed to stat blob")
		return manifest.Descriptor{}, err
	}

	if fi.IsDir() {
		log.Ctx(ctx).Warn().
			Str("method", "globalBlobStore.Stat").
			Str("path", path).
			Msg("blob path should not be a directory")
		return manifest.Descriptor{}, ErrBlobUnknown
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobStore.Stat").
		Str("path", path).
		Int64("size", fi.Size()).
		Msg("blob stat completed")

	return manifest.Descriptor{
		Size:      fi.Size(),
		MediaType: "application/octet-stream",
		Digest:    dgst,
	}, nil
}

func GlobalPathFn(dgst digest.Digest) (string, error) {
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
