// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/types"

	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
)

// GetBlob returns the blob for the given object sha.
func (g Adapter) GetBlob(ctx context.Context, repoPath string, sha string, sizeLimit int64) (*types.BlobReader, error) {
	repo, err := g.repoProvider.Get(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	blob, err := repo.BlobObject(gogitplumbing.NewHash(sha))
	if err != nil {
		if err == gogitplumbing.ErrObjectNotFound {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get blob object: %w", err)
	}

	objectSize := blob.Size
	contentSize := objectSize
	if sizeLimit > 0 && contentSize > sizeLimit {
		contentSize = sizeLimit
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to open blob object: %w", err)
	}

	return &types.BlobReader{
		SHA:         sha,
		Size:        objectSize,
		ContentSize: contentSize,
		Content:     LimitReadCloser(reader, contentSize),
	}, nil
}

func LimitReadCloser(r io.ReadCloser, n int64) io.ReadCloser {
	return limitReadCloser{
		r: io.LimitReader(r, n),
		c: r,
	}
}

// limitReadCloser implements io.ReadCloser interface.
type limitReadCloser struct {
	r io.Reader
	c io.Closer
}

func (l limitReadCloser) Read(p []byte) (n int, err error) {
	return l.r.Read(p)
}

func (l limitReadCloser) Close() error {
	return l.c.Close()
}
