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
	if !bw.resumableDigestEnabled {
		return errResumableDigestNotAvailable
	}

	h, ok := bw.digester.Hash().(encoding.BinaryUnmarshaler)
	if !ok {
		return errResumableDigestNotAvailable
	}

	offset := bw.fileWriter.Size()
	if offset == bw.written {
		// State of digester is already at the requested offset.
		return nil
	}

	// List hash states from storage backend.
	var hashStateMatch hashStateEntry
	hashStates, err := bw.getStoredHashStates(ctx)
	if err != nil {
		return fmt.Errorf("unable to get stored hash states with offset %d: %w", offset, err)
	}

	// Find the highest stored hashState with offset equal to
	// the requested offset.
	for _, hashState := range hashStates {
		if hashState.offset == offset {
			hashStateMatch = hashState
			break // Found an exact offset match.
		}
	}

	if hashStateMatch.offset == 0 {
		// No need to load any state, just reset the hasher.
		h.(hash.Hash).Reset() //nolint:errcheck
	} else {
		storedState, err := bw.driver.GetContent(ctx, hashStateMatch.path)
		if err != nil {
			return err
		}

		if err = h.UnmarshalBinary(storedState); err != nil {
			return err
		}
		bw.written = hashStateMatch.offset
	}

	// Mind the gap.
	if gapLen := offset - bw.written; gapLen > 0 {
		return errResumableDigestNotAvailable
	}

	return nil
}

// getStoredHashStates returns a slice of hashStateEntries for this upload.
func (bw *globalBlobWriter) getStoredHashStates(ctx context.Context) ([]hashStateEntry, error) {
	uploadHashStatePathPrefix, err := pathFor(
		globalUploadHashStatePathSpec{
			id:   bw.id,
			alg:  bw.digester.Digest().Algorithm(),
			list: true,
		},
	)
	if err != nil {
		return nil, err
	}

	paths, err := bw.globalBlobStore.driver.List(ctx, uploadHashStatePathPrefix)
	if err != nil {
		if ok := errors.As(err, &storagedriver.PathNotFoundError{}); !ok {
			return nil, err
		}
		// Treat PathNotFoundError as no entries.
		paths = nil
	}

	hashStateEntries := make([]hashStateEntry, 0, len(paths))

	for _, p := range paths {
		pathSuffix := path.Base(p)
		// The suffix should be the offset.
		offset, err := strconv.ParseInt(pathSuffix, 0, 64)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("unable to parse offset from upload state path %q: %s", p, err)
		}

		hashStateEntries = append(hashStateEntries, hashStateEntry{offset: offset, path: p})
	}

	return hashStateEntries, nil
}

func (bw *globalBlobWriter) storeHashState(ctx context.Context) error {
	if !bw.resumableDigestEnabled {
		return errResumableDigestNotAvailable
	}

	h, ok := bw.digester.Hash().(encoding.BinaryMarshaler)
	if !ok {
		return errResumableDigestNotAvailable
	}

	state, err := h.MarshalBinary()
	if err != nil {
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
		return err
	}

	return bw.driver.PutContent(ctx, uploadHashStatePath, state)
}
