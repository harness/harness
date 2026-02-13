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
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// blobWriter is used to control the various aspects of resumable
// blob upload.
type globalBlobWriter struct {
	ctx             context.Context
	globalBlobStore *globalBlobStore

	id       string
	digester digest.Digester
	written  int64 // track the write to digester

	fileWriter driver.FileWriter
	driver     driver.StorageDriver
	path       string

	resumableDigestEnabled bool
	committed              bool
	// For Global Blob Store
	isMultiPart bool
}

var _ BlobWriter = &globalBlobWriter{}

// ID returns the identifier for this upload.
func (bw *globalBlobWriter) ID() string {
	return bw.id
}

// Commit marks the upload as completed, returning a valid descriptor. The
// final size and digest are checked against the first descriptor provided.
func (bw *globalBlobWriter) Commit(ctx context.Context, pathPrefix string, desc manifest.Descriptor) (
	manifest.Descriptor, error,
) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.Commit").
		Str("upload_id", bw.id).
		Str("path_prefix", pathPrefix).
		Str("digest", desc.Digest.String()).
		Int64("desc_size", desc.Size).
		Msg("starting blob commit")

	if err := bw.fileWriter.Commit(ctx); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Commit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to commit file writer")
		return manifest.Descriptor{}, err
	}

	if err := bw.Close(); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Commit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to close blob writer")
	}
	desc.Size = bw.Size()

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.Commit").
		Str("upload_id", bw.id).
		Int64("final_size", desc.Size).
		Msg("validating blob")

	canonical, err := bw.validateBlob(ctx, desc)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Commit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("blob validation failed")
		return manifest.Descriptor{}, err
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.Commit").
		Str("upload_id", bw.id).
		Str("canonical_digest", canonical.Digest.String()).
		Msg("moving blob to permanent location")

	if err := bw.moveBlob(ctx, pathPrefix, canonical); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Commit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to move blob")
		return manifest.Descriptor{}, err
	}

	if err := bw.removeResources(ctx); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Commit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to remove upload resources")
		return manifest.Descriptor{}, err
	}

	bw.committed = true
	return canonical, nil
}

// PlainCommit commits the files and move to desired location without any validity.
// To be deprecated SOON after global storage takes over.
func (bw *globalBlobWriter) PlainCommit(ctx context.Context, sha256 string) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.PlainCommit").
		Str("upload_id", bw.id).
		Msg("starting plain commit")

	if err := bw.fileWriter.Commit(ctx); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.PlainCommit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to commit file writer")
		return err
	}

	err := bw.Close()
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.PlainCommit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to close blob writer")
		return err
	}

	err = bw.globalBlobStore.move(ctx, bw.id, sha256)
	if err != nil {
		log.Ctx(ctx).Error().
			Str("method", "globalBlobWriter.PlainCommit").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to move file to permanent location")
		return fmt.Errorf("failed to Move the file on permanent location for sha256: %s %w", sha256, err)
	}
	return nil
}

// Cancel the blob upload process, releasing any resources associated with
// the writer and canceling the operation.
func (bw *globalBlobWriter) Cancel(ctx context.Context) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.Cancel").
		Str("upload_id", bw.id).
		Msg("canceling blob upload")

	if err := bw.fileWriter.Cancel(ctx); err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Cancel").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to cancel file writer")
		return err
	}

	if err := bw.Close(); err != nil {
		log.Ctx(ctx).Error().
			Str("method", "globalBlobWriter.Cancel").
			Str("upload_id", bw.id).
			Err(err).
			Msg("error closing blob writer during cancel")
	}

	err := bw.removeResources(ctx)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.Cancel").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to remove resources during cancel")
		return err
	}

	return nil
}

func (bw *globalBlobWriter) Size() int64 {
	return bw.fileWriter.Size()
}

func (bw *globalBlobWriter) Write(p []byte) (int, error) {
	log.Ctx(bw.ctx).Debug().
		Str("method", "globalBlobWriter.Write").
		Str("upload_id", bw.id).
		Int("chunk_size", len(p)).
		Msg("writing data chunk")

	// We don't support multipart uploads in generic.
	if !bw.isMultiPart {
		n, err := bw.fileWriter.Write(p)
		if err != nil {
			log.Ctx(bw.ctx).Debug().
				Str("method", "globalBlobWriter.Write").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to write to file writer")
		}
		return n, err
	}

	// Ensure that the current write offset matches how many bytes have been
	// written to the digester. If not, we need to update the digest state to
	// match the current write position.
	if err := bw.resumeDigest(bw.globalBlobStore.ctx); err != nil && !errors.Is(err, errResumableDigestNotAvailable) {
		log.Ctx(bw.ctx).Debug().
			Str("method", "globalBlobWriter.Write").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to resume digest")
		return 0, err
	}

	_, err := bw.fileWriter.Write(p)
	if err != nil {
		log.Ctx(bw.ctx).Debug().
			Str("method", "globalBlobWriter.Write").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to write to file writer")
		return 0, err
	}

	n, err := bw.digester.Hash().Write(p)
	bw.written += int64(n)

	if err != nil {
		log.Ctx(bw.ctx).Debug().
			Str("method", "globalBlobWriter.Write").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to write to digester")
	}

	return n, err
}

func (bw *globalBlobWriter) Close() error {
	log.Ctx(bw.ctx).Debug().
		Str("method", "globalBlobWriter.Close").
		Str("upload_id", bw.id).
		Bool("committed", bw.committed).
		Msg("closing blob writer")

	// We don't support multipart uploads in Generic
	if !bw.isMultiPart {
		err := bw.fileWriter.Close()
		if err != nil {
			log.Ctx(bw.ctx).Debug().
				Str("method", "globalBlobWriter.Close").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to close file writer")
		}
		return err
	}

	if bw.committed {
		log.Ctx(bw.ctx).Debug().
			Str("method", "globalBlobWriter.Close").
			Str("upload_id", bw.id).
			Msg("attempted to close already committed blob writer")
		return errors.New("blobwriter close after commit")
	}

	if err := bw.storeHashState(bw.globalBlobStore.ctx); err != nil && !errors.Is(err, errResumableDigestNotAvailable) {
		log.Ctx(bw.ctx).Debug().
			Str("method", "globalBlobWriter.Close").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to store hash state")
		return err
	}

	return bw.fileWriter.Close()
}

// validateBlob checks the data against the digest, returning an error if it
// does not match. The canonical descriptor is returned.
func (bw *globalBlobWriter) validateBlob(ctx context.Context, desc manifest.Descriptor) (manifest.Descriptor, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.validateBlob").
		Str("upload_id", bw.id).
		Str("digest", desc.Digest.String()).
		Int64("desc_size", desc.Size).
		Msg("starting blob validation")

	var (
		verified, fullHash bool
		canonical          digest.Digest
	)

	if desc.Digest == "" {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Msg("cannot validate against empty digest")
		return manifest.Descriptor{}, BlobInvalidDigestError{
			Reason: fmt.Errorf("cannot validate against empty digest"),
		}
	}

	var size int64

	// Stat the on disk file
	if fi, err := bw.driver.Stat(ctx, bw.path); err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Str("path", bw.path).
				Msg("file not found, setting size to 0")
			desc.Size = 0
		} else {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to stat upload file")
			return manifest.Descriptor{}, err
		}
	} else {
		if fi.IsDir() {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Str("path", bw.path).
				Msg("unexpected directory at upload location")
			return manifest.Descriptor{}, fmt.Errorf("unexpected directory at upload location %q", bw.path)
		}

		size = fi.Size()
	}

	if desc.Size > 0 {
		if desc.Size != size {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Int64("expected_size", desc.Size).
				Int64("actual_size", size).
				Msg("blob size mismatch")
			return manifest.Descriptor{}, ErrBlobInvalidLength
		}
	} else {
		desc.Size = size
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.validateBlob").
		Str("upload_id", bw.id).
		Int64("size", size).
		Msg("attempting to resume digest for validation")

	if err := bw.resumeDigest(ctx); err == nil {
		canonical = bw.digester.Digest()

		if canonical.Algorithm() == desc.Digest.Algorithm() {
			verified = desc.Digest == canonical
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Bool("verified", verified).
				Msg("digest verification using resumed digester")
		} else {
			fullHash = true
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Msg("algorithm mismatch, need full hash")
		}
	} else if errors.Is(err, errResumableDigestNotAvailable) {
		fullHash = true
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Msg("resumable digest not available, need full hash")
	} else {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to resume digest")
		return manifest.Descriptor{}, err
	}

	if fullHash && bw.written == size && digest.Canonical == desc.Digest.Algorithm() {
		canonical = bw.digester.Digest()
		verified = desc.Digest == canonical
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Bool("verified", verified).
			Msg("using optimization: written matches size")
	}

	if fullHash && !verified {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Msg("performing full hash verification from storage")

		digester := digest.Canonical.Digester()
		verifier := desc.Digest.Verifier()

		fr, err := NewFileReader(ctx, bw.driver, bw.path, desc.Size)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to create file reader for verification")
			return manifest.Descriptor{}, err
		}
		defer fr.Close()

		tr := io.TeeReader(fr, digester.Hash())

		if _, err := io.Copy(verifier, tr); err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.validateBlob").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to copy data for verification")
			return manifest.Descriptor{}, err
		}

		canonical = digester.Digest()
		verified = verifier.Verified()
	}

	if !verified {
		log.Ctx(ctx).Error().
			Str("method", "globalBlobWriter.validateBlob").
			Str("upload_id", bw.id).
			Str("canonical_digest", canonical.String()).
			Str("provided_digest", desc.Digest.String()).
			Msg("canonical digest does not match provided digest")
		return manifest.Descriptor{}, BlobInvalidDigestError{
			Digest: desc.Digest,
			Reason: fmt.Errorf("content does not match digest"),
		}
	}

	desc.Digest = canonical

	if desc.MediaType == "" {
		desc.MediaType = "application/octet-stream"
	}

	return desc, nil
}

// moveBlob moves the data into its final, hash-qualified destination,
// identified by dgst. The layer should be validated before commencing the
// move.
func (bw *globalBlobWriter) moveBlob(ctx context.Context, _ string, desc manifest.Descriptor) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.moveBlob").
		Str("upload_id", bw.id).
		Str("digest", desc.Digest.String()).
		Msg("starting blob move")

	blobPath, err := pathFor(
		globalBlobPathSpec{
			digest: desc.Digest,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.moveBlob").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get blob path")
		return err
	}

	// Check for existence
	if _, err := bw.globalBlobStore.driver.Stat(ctx, blobPath); err != nil {
		if !errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.moveBlob").
				Str("upload_id", bw.id).
				Str("blob_path", blobPath).
				Err(err).
				Msg("failed to stat blob path")
			return err
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.moveBlob").
			Str("upload_id", bw.id).
			Str("blob_path", blobPath).
			Msg("blob path does not exist, proceeding with move")
	} else {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.moveBlob").
			Str("upload_id", bw.id).
			Str("blob_path", blobPath).
			Msg("blob already exists at destination, skipping move")
		return nil
	}

	// If no data was received, we may not actually have a file on disk. Check
	// the size here and write a zero-length file to blobPath if this is the
	// case. For the most part, this should only ever happen with zero-length
	// blobs.
	if _, err := bw.globalBlobStore.driver.Stat(ctx, bw.path); err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			if desc.Digest == digestSha256Empty {
				log.Ctx(ctx).Debug().
					Str("method", "globalBlobWriter.moveBlob").
					Str("upload_id", bw.id).
					Msg("writing empty blob for empty digest")
				return bw.globalBlobStore.driver.PutContent(ctx, blobPath, []byte{})
			}

			log.Ctx(ctx).Warn().
				Str("method", "globalBlobWriter.moveBlob").
				Str("upload_id", bw.id).
				Str("digest", desc.Digest.String()).
				Msg("attempted to move zero-length content with non-zero digest")
		} else {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.moveBlob").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to stat upload path")
			return err
		}
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.moveBlob").
		Str("upload_id", bw.id).
		Str("src_path", bw.path).
		Str("dst_path", blobPath).
		Msg("moving blob to permanent location")

	err = bw.globalBlobStore.driver.Move(ctx, bw.path, blobPath)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.moveBlob").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to move blob")
		return err
	}
	return nil
}

// removeResources should clean up all resources associated with the upload
// instance. An error will be returned if the clean up cannot proceed. If the
// resources are already not present, no error will be returned.
func (bw *globalBlobWriter) removeResources(ctx context.Context) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.removeResources").
		Str("upload_id", bw.id).
		Msg("removing upload resources")

	dataPath, err := pathFor(
		globalUploadDataPathSpec{
			id: bw.id,
		},
	)
	if err != nil {
		log.Ctx(ctx).Error().
			Str("method", "globalBlobWriter.removeResources").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get data path")
		return err
	}

	// Resolve and delete the containing directory, which should include any
	// upload related files.
	dirPath := path.Dir(dataPath)
	if err := bw.globalBlobStore.driver.Delete(ctx, dirPath); err != nil {
		if !errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(ctx).Error().
				Str("method", "globalBlobWriter.removeResources").
				Str("upload_id", bw.id).
				Str("dir_path", dirPath).
				Err(err).
				Msg("unable to delete layer upload resources")
			return err
		}
	}

	return nil
}

func (bw *globalBlobWriter) Reader() (io.ReadCloser, error) {
	log.Ctx(bw.ctx).Debug().
		Str("method", "globalBlobWriter.Reader").
		Str("upload_id", bw.id).
		Str("path", bw.path).
		Msg("getting reader for uploaded data")

	try := 1
	for try <= 5 {
		_, err := bw.driver.Stat(bw.ctx, bw.path)
		if err == nil {
			log.Ctx(bw.ctx).Debug().
				Str("method", "globalBlobWriter.Reader").
				Str("upload_id", bw.id).
				Int("try", try).
				Msg("file found")
			break
		}
		if errors.As(err, &driver.PathNotFoundError{}) {
			log.Ctx(bw.ctx).Debug().
				Str("method", "globalBlobWriter.Reader").
				Str("upload_id", bw.id).
				Int("try", try).
				Msg("file not found, retrying after sleep")
			time.Sleep(1 * time.Second)
			try++
		} else {
			log.Ctx(bw.ctx).Debug().
				Str("method", "globalBlobWriter.Reader").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to stat file")
			return nil, err
		}
	}

	readCloser, err := bw.driver.Reader(bw.ctx, bw.path, 0)
	if err != nil {
		log.Ctx(bw.ctx).Error().
			Str("method", "globalBlobWriter.Reader").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get reader")
		return nil, err
	}
	return readCloser, nil
}
