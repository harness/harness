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
	"path"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

var errResumableDigestNotAvailable = errors.New("resumable digest not available")

const (
	// digestSha256Empty is the canonical sha256 digest of empty data.
	digestSha256Empty = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// blobWriter is used to control the various aspects of resumable
// blob upload.
type blobWriter struct {
	ctx       context.Context
	blobStore *ociBlobStore

	id       string
	digester digest.Digester
	written  int64 // track the write to digester

	fileWriter driver.FileWriter
	driver     driver.StorageDriver
	path       string

	resumableDigestEnabled bool
	committed              bool
}

var _ BlobWriter = &blobWriter{}

// ID returns the identifier for this upload.
func (bw *blobWriter) ID() string {
	return bw.id
}

// Commit marks the upload as completed, returning a valid descriptor. The
// final size and digest are checked against the first descriptor provided.
func (bw *blobWriter) Commit(ctx context.Context, pathPrefix string, desc manifest.Descriptor) (
	manifest.Descriptor, error,
) {
	dcontext.GetLogger(ctx, log.Debug()).Msg("(*blobWriter).Commit")

	if err := bw.fileWriter.Commit(ctx); err != nil {
		return manifest.Descriptor{}, err
	}

	bw.Close()
	desc.Size = bw.Size()

	canonical, err := bw.validateBlob(ctx, desc)
	if err != nil {
		return manifest.Descriptor{}, err
	}

	if err := bw.moveBlob(ctx, pathPrefix, canonical); err != nil {
		return manifest.Descriptor{}, err
	}

	if err := bw.removeResources(ctx); err != nil {
		return manifest.Descriptor{}, err
	}

	bw.committed = true
	return canonical, nil
}

// Cancel the blob upload process, releasing any resources associated with
// the writer and canceling the operation.
func (bw *blobWriter) Cancel(ctx context.Context) error {
	dcontext.GetLogger(ctx, log.Debug()).Msg("(*blobWriter).Cancel")
	if err := bw.fileWriter.Cancel(ctx); err != nil {
		return err
	}

	if err := bw.Close(); err != nil {
		dcontext.GetLogger(ctx, log.Error()).Msgf("error closing blobwriter: %s", err)
	}

	return bw.removeResources(ctx)
}

func (bw *blobWriter) Size() int64 {
	return bw.fileWriter.Size()
}

func (bw *blobWriter) Write(p []byte) (int, error) {
	// Ensure that the current write offset matches how many bytes have been
	// written to the digester. If not, we need to update the digest state to
	// match the current write position.
	if err := bw.resumeDigest(bw.blobStore.ctx); err != nil && !errors.Is(err, errResumableDigestNotAvailable) {
		return 0, err
	}

	_, err := bw.fileWriter.Write(p)
	if err != nil {
		return 0, err
	}

	n, err := bw.digester.Hash().Write(p)
	bw.written += int64(n)

	return n, err
}

func (bw *blobWriter) Close() error {
	if bw.committed {
		return errors.New("blobwriter close after commit")
	}

	if err := bw.storeHashState(bw.blobStore.ctx); err != nil && !errors.Is(err, errResumableDigestNotAvailable) {
		return err
	}

	return bw.fileWriter.Close()
}

// validateBlob checks the data against the digest, returning an error if it
// does not match. The canonical descriptor is returned.
func (bw *blobWriter) validateBlob(ctx context.Context, desc manifest.Descriptor) (manifest.Descriptor, error) {
	var (
		verified, fullHash bool
		canonical          digest.Digest
	)

	if desc.Digest == "" {
		// if no descriptors are provided, we have nothing to validate
		// against. We don't really want to support this for the registry.
		return manifest.Descriptor{}, BlobInvalidDigestError{
			Reason: fmt.Errorf("cannot validate against empty digest"),
		}
	}

	var size int64

	// Stat the on disk file
	if fi, err := bw.driver.Stat(ctx, bw.path); err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			desc.Size = 0
		} else {
			// Any other error we want propagated up the stack.
			return manifest.Descriptor{}, err
		}
	} else {
		if fi.IsDir() {
			return manifest.Descriptor{}, fmt.Errorf("unexpected directory at upload location %q", bw.path)
		}

		size = fi.Size()
	}

	if desc.Size > 0 {
		if desc.Size != size {
			return manifest.Descriptor{}, ErrBlobInvalidLength
		}
	} else {
		// if provided 0 or negative length, we can assume caller doesn't know or
		// care about length.
		desc.Size = size
	}

	if err := bw.resumeDigest(ctx); err == nil {
		canonical = bw.digester.Digest()

		if canonical.Algorithm() == desc.Digest.Algorithm() {
			// Common case: client and server prefer the same canonical digest
			// algorithm - currently SHA256.
			verified = desc.Digest == canonical
		} else {
			// The client wants to use a different digest algorithm. They'll just
			// have to be patient and wait for us to download and re-hash the
			// uploaded content using that digest algorithm.
			fullHash = true
		}
	} else if errors.Is(err, errResumableDigestNotAvailable) {
		// Not using resumable digests, so we need to hash the entire layer.
		fullHash = true
	} else {
		return manifest.Descriptor{}, err
	}

	if fullHash && bw.written == size && digest.Canonical == desc.Digest.Algorithm() {
		// a fantastic optimization: if the the written data and the size are
		// the same, we don't need to read the data from the backend. This is
		// because we've written the entire file in the lifecycle of the
		// current instance.
		canonical = bw.digester.Digest()
		verified = desc.Digest == canonical
	}

	if fullHash && !verified {
		// If the check based on size fails, we fall back to the slowest of
		// paths. We may be able to make the size-based check a stronger
		// guarantee, so this may be defensive.
		digester := digest.Canonical.Digester()
		verifier := desc.Digest.Verifier()

		// Read the file from the backend Driver and validate it.
		fr, err := NewFileReader(ctx, bw.driver, bw.path, desc.Size)
		if err != nil {
			return manifest.Descriptor{}, err
		}
		defer fr.Close()

		tr := io.TeeReader(fr, digester.Hash())

		if _, err := io.Copy(verifier, tr); err != nil {
			return manifest.Descriptor{}, err
		}

		canonical = digester.Digest()
		verified = verifier.Verified()
	}
	if !verified {
		dcontext.GetLoggerWithFields(
			ctx, log.Ctx(ctx).Error(),
			map[any]any{
				"canonical": canonical,
				"provided":  desc.Digest,
			}, "canonical", "provided",
		).
			Msg("canonical digest does match provided digest")
		return manifest.Descriptor{}, BlobInvalidDigestError{
			Digest: desc.Digest,
			Reason: fmt.Errorf("content does not match digest"),
		}
	}

	// update desc with canonical hash
	desc.Digest = canonical

	if desc.MediaType == "" {
		desc.MediaType = "application/octet-stream"
	}

	return desc, nil
}

// moveBlob moves the data into its final, hash-qualified destination,
// identified by dgst. The layer should be validated before commencing the
// move.
func (bw *blobWriter) moveBlob(ctx context.Context, pathPrefix string, desc manifest.Descriptor) error {
	blobPath, err := pathFor(
		blobDataPathSpec{
			digest: desc.Digest,
			path:   pathPrefix,
		},
	)
	if err != nil {
		return err
	}

	// Check for existence
	if _, err := bw.blobStore.driver.Stat(ctx, blobPath); err != nil {
		log.Ctx(ctx).Info().Msgf("Error type: %T, value: %v\n", err, err)
		if !errors.As(err, &driver.PathNotFoundError{}) {
			return err
		}
	} else {
		// If the path exists, we can assume that the content has already
		// been uploaded, since the blob storage is content-addressable.
		// While it may be corrupted, detection of such corruption belongs
		// elsewhere.
		return nil
	}

	// If no data was received, we may not actually have a file on disk. Check
	// the size here and write a zero-length file to blobPath if this is the
	// case. For the most part, this should only ever happen with zero-length
	// blobs.
	if _, err := bw.blobStore.driver.Stat(ctx, bw.path); err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			if desc.Digest == digestSha256Empty {
				return bw.blobStore.driver.PutContent(ctx, blobPath, []byte{})
			}

			// We let this fail during the move below.
			log.Ctx(ctx).Warn().
				Interface("upload.id", bw.ID()).
				Interface("digest", desc.Digest).
				Msg("attempted to move zero-length content with non-zero digest")
		} else {
			return err // unrelated error
		}
	}

	return bw.blobStore.driver.Move(ctx, bw.path, blobPath)
}

// removeResources should clean up all resources associated with the upload
// instance. An error will be returned if the clean up cannot proceed. If the
// resources are already not present, no error will be returned.
func (bw *blobWriter) removeResources(ctx context.Context) error {
	dataPath, err := pathFor(
		uploadDataPathSpec{
			path:     bw.blobStore.rootParentRef,
			repoName: bw.blobStore.repoKey,
			id:       bw.id,
		},
	)
	if err != nil {
		return err
	}

	// Resolve and delete the containing directory, which should include any
	// upload related files.
	dirPath := path.Dir(dataPath)
	if err := bw.blobStore.driver.Delete(ctx, dirPath); err != nil {
		if !errors.As(err, &driver.PathNotFoundError{}) {
			// This should be uncommon enough such that returning an error
			// should be okay. At this point, the upload should be mostly
			// complete, but perhaps the backend became unaccessible.
			dcontext.GetLogger(ctx, log.Error()).Msgf("unable to delete layer upload resources %q: %v", dirPath, err)
			return err
		}
	}

	return nil
}

func (bw *blobWriter) Reader() (io.ReadCloser, error) {
	try := 1
	for try <= 5 {
		_, err := bw.driver.Stat(bw.ctx, bw.path)
		if err == nil {
			break
		}
		if errors.As(err, &driver.PathNotFoundError{}) {
			dcontext.GetLogger(bw.ctx, log.Debug()).Msgf("Nothing found on try %d, sleeping...", try)
			time.Sleep(1 * time.Second)
			try++
		} else {
			return nil, err
		}
	}

	readCloser, err := bw.driver.Reader(bw.ctx, bw.path, 0)
	if err != nil {
		return nil, err
	}

	return readCloser, nil
}
