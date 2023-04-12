// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/types"

	"code.gitea.io/gitea/modules/git"
)

// GetBlob returns the blob for the given object sha.
func (g Adapter) GetBlob(ctx context.Context, repoPath string, sha string, sizeLimit int64) (*types.BlobReader, error) {
	// Note: We are avoiding gitea blob implementation, as that is tied to the lifetime of the repository object.
	// Instead, we just use the gitea helper methods ourselves.
	stdIn, stdOut, cancel := git.CatFileBatch(ctx, repoPath)

	_, err := stdIn.Write([]byte(sha + "\n"))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to write blob sha to git stdin: %w", err)
	}

	objectSHA, objectType, objectSize, err := git.ReadBatchLine(stdOut)
	if err != nil {
		cancel()
		return nil, processGiteaErrorf(err, "failed to read cat-file batch line")
	}

	if string(objectSHA) != sha {
		cancel()
		return nil, fmt.Errorf("cat-file returned object sha '%s' but expected '%s'", objectSHA, sha)
	}
	if objectType != string(git.ObjectBlob) {
		cancel()
		return nil, fmt.Errorf("cat-file returned object type '%s' but expected '%s'", objectType, git.ObjectBlob)
	}

	contentSize := objectSize
	if sizeLimit > 0 && sizeLimit < contentSize {
		contentSize = sizeLimit
	}

	return &types.BlobReader{
		SHA:         sha,
		Size:        objectSize,
		ContentSize: contentSize,
		Content: &exactLimitReader{
			reader:         stdOut,
			remainingBytes: contentSize,
			close: func() error {
				// TODO: is there a better (but short) way to clear the buffer?
				// gitea is .Discard()'ing elements here until it's empty.
				stdOut.Reset(bytes.NewBuffer([]byte{}))
				cancel()
				return nil
			},
		},
	}, nil
}

// exactLimitReader reads the content of a reader and ensures no more than the specified bytes will be requested from
// the underlaying reader. This is required for readers that don't ensure completion after reading all remaining bytes.
// io.LimitReader doesn't work as it waits for bytes that never come, io.SectionReader would requrie an io.ReaderAt.
type exactLimitReader struct {
	reader         io.Reader
	remainingBytes int64
	close          func() error
}

func (r *exactLimitReader) Read(p []byte) (int, error) {
	if r.remainingBytes <= 0 {
		return 0, io.EOF
	}

	if int64(len(p)) > r.remainingBytes {
		p = p[0:r.remainingBytes]
	}
	n, err := r.reader.Read(p)
	r.remainingBytes -= int64(n)

	return n, err
}

func (r *exactLimitReader) Close() error {
	return r.close()
}
