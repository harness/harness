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
	"encoding"
	"errors"
	"fmt"
	"hash"
	"path"
	"strconv"

	storagedriver "github.com/harness/gitness/registry/app/driver"

	"github.com/rs/zerolog/log"
)

// resumeDigest attempts to restore the state of the internal hash function
// by loading the most recent saved hash state equal to the current size of the blob.
func (bw *globalBlobWriter) resumeDigest(ctx context.Context) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.resumeDigest").
		Str("upload_id", bw.id).
		Msg("attempting to resume digest")

	if !bw.resumableDigestEnabled {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Msg("resumable digest not enabled")
		return errResumableDigestNotAvailable
	}

	h, ok := bw.digester.Hash().(encoding.BinaryUnmarshaler)
	if !ok {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Msg("hash does not support binary unmarshaling")
		return errResumableDigestNotAvailable
	}

	offset := bw.fileWriter.Size()
	if offset == bw.written {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Int64("offset", offset).
			Msg("digester already at requested offset")
		return nil
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.resumeDigest").
		Str("upload_id", bw.id).
		Int64("current_offset", offset).
		Int64("written", bw.written).
		Msg("loading stored hash states")

	// List hash states from storage backend.
	var hashStateMatch hashStateEntry
	hashStates, err := bw.getStoredHashStates(ctx)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get stored hash states")
		return fmt.Errorf("unable to get stored hash states with offset %d: %w", offset, err)
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.resumeDigest").
		Str("upload_id", bw.id).
		Int("hash_states_count", len(hashStates)).
		Msg("searching for matching hash state")

	// Find the highest stored hashState with offset equal to
	// the requested offset.
	for _, hashState := range hashStates {
		if hashState.offset == offset {
			hashStateMatch = hashState
			break // Found an exact offset match.
		}
	}

	if hashStateMatch.offset == 0 {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Msg("no matching hash state found, resetting hasher")
		h.(hash.Hash).Reset() //nolint:errcheck
	} else {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Int64("match_offset", hashStateMatch.offset).
			Str("match_path", hashStateMatch.path).
			Msg("loading stored hash state")

		storedState, err := bw.driver.GetContent(ctx, hashStateMatch.path)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.resumeDigest").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to get stored hash state content")
			return err
		}

		if err = h.UnmarshalBinary(storedState); err != nil {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.resumeDigest").
				Str("upload_id", bw.id).
				Err(err).
				Msg("failed to unmarshal hash state")
			return err
		}
		bw.written = hashStateMatch.offset
	}

	// Mind the gap.
	if gapLen := offset - bw.written; gapLen > 0 {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.resumeDigest").
			Str("upload_id", bw.id).
			Int64("gap_length", gapLen).
			Msg("gap detected between offset and written, cannot resume")
		return errResumableDigestNotAvailable
	}

	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.resumeDigest").
		Str("upload_id", bw.id).
		Int64("written", bw.written).
		Msg("digest resumed successfully")
	return nil
}

// getStoredHashStates returns a slice of hashStateEntries for this upload.
func (bw *globalBlobWriter) getStoredHashStates(ctx context.Context) ([]hashStateEntry, error) {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.getStoredHashStates").
		Str("upload_id", bw.id).
		Str("algorithm", string(bw.digester.Digest().Algorithm())).
		Msg("getting stored hash states")

	uploadHashStatePathPrefix, err := pathFor(
		globalUploadHashStatePathSpec{
			id:   bw.id,
			alg:  bw.digester.Digest().Algorithm(),
			list: true,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.getStoredHashStates").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get hash state path prefix")
		return nil, err
	}

	paths, err := bw.globalBlobStore.driver.List(ctx, uploadHashStatePathPrefix)
	if err != nil {
		if ok := errors.As(err, &storagedriver.PathNotFoundError{}); !ok {
			log.Ctx(ctx).Debug().
				Str("method", "globalBlobWriter.getStoredHashStates").
				Str("upload_id", bw.id).
				Str("path_prefix", uploadHashStatePathPrefix).
				Err(err).
				Msg("failed to list hash states")
			return nil, err
		}
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.getStoredHashStates").
			Str("upload_id", bw.id).
			Msg("no hash states found (path not found)")
		paths = nil
	}

	hashStateEntries := make([]hashStateEntry, 0, len(paths))

	for _, p := range paths {
		pathSuffix := path.Base(p)
		// The suffix should be the offset.
		offset, err := strconv.ParseInt(pathSuffix, 0, 64)
		if err != nil {
			log.Ctx(ctx).Error().
				Str("method", "globalBlobWriter.getStoredHashStates").
				Str("upload_id", bw.id).
				Str("path", p).
				Err(err).
				Msg("unable to parse offset from upload state path")
			continue
		}

		hashStateEntries = append(hashStateEntries, hashStateEntry{offset: offset, path: p})
	}

	return hashStateEntries, nil
}

func (bw *globalBlobWriter) storeHashState(ctx context.Context) error {
	log.Ctx(ctx).Debug().
		Str("method", "globalBlobWriter.storeHashState").
		Str("upload_id", bw.id).
		Bool("resumable_enabled", bw.resumableDigestEnabled).
		Int64("written", bw.written).
		Msg("storing hash state")

	if !bw.resumableDigestEnabled {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.storeHashState").
			Str("upload_id", bw.id).
			Msg("resumable digest not enabled")
		return errResumableDigestNotAvailable
	}

	h, ok := bw.digester.Hash().(encoding.BinaryMarshaler)
	if !ok {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.storeHashState").
			Str("upload_id", bw.id).
			Msg("hash does not support binary marshaling")
		return errResumableDigestNotAvailable
	}

	state, err := h.MarshalBinary()
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.storeHashState").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to marshal hash state")
		return err
	}

	uploadHashStatePath, err := pathFor(
		globalUploadHashStatePathSpec{
			id:     bw.id,
			alg:    bw.digester.Digest().Algorithm(),
			offset: bw.written,
		},
	)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.storeHashState").
			Str("upload_id", bw.id).
			Err(err).
			Msg("failed to get hash state path")
		return err
	}

	err = bw.driver.PutContent(ctx, uploadHashStatePath, state)
	if err != nil {
		log.Ctx(ctx).Debug().
			Str("method", "globalBlobWriter.storeHashState").
			Str("upload_id", bw.id).
			Str("path", uploadHashStatePath).
			Err(err).
			Msg("failed to store hash state")
		return err
	}

	return nil
}
