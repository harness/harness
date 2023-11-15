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

package adapter

import (
	"context"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
)

// GetBlob returns the blob for the given object sha.
func (a Adapter) GetBlob(
	ctx context.Context,
	repoPath string,
	sha string,
	sizeLimit int64,
) (*types.BlobReader, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	repo, err := a.repoProvider.Get(ctx, repoPath)
	if err != nil {
		return nil, errors.Internal("failed to open repository", err)
	}

	blob, err := repo.BlobObject(gogitplumbing.NewHash(sha))
	if err != nil {
		if errors.Is(err, gogitplumbing.ErrObjectNotFound) {
			return nil, errors.NotFound("blob sha %s not found", sha)
		}
		return nil, errors.Internal("failed to get blob object for sha '%s'", sha, err)
	}

	objectSize := blob.Size
	contentSize := objectSize
	if sizeLimit > 0 && contentSize > sizeLimit {
		contentSize = sizeLimit
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, errors.Internal("failed to open blob object for sha '%s'", sha, err)
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
