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
	"fmt"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"
)

// GetBlob returns the blob for the given object sha.
func (a Adapter) GetBlob(
	ctx context.Context,
	repoPath string,
	sha string,
	sizeLimit int64,
) (*types.BlobReader, error) {
	stdIn, stdOut, cancel := CatFileBatch(ctx, repoPath)

	_, err := stdIn.Write([]byte(sha + "\n"))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to write blob sha to git stdin: %w", err)
	}

	objectSHA, objectType, objectSize, err := ReadBatchHeaderLine(stdOut)
	if err != nil {
		cancel()
		return nil, processGiteaErrorf(err, "failed to read cat-file batch line")
	}

	if string(objectSHA) != sha {
		cancel()
		return nil, fmt.Errorf("cat-file returned object sha '%s' but expected '%s'", objectSHA, sha)
	}
	if objectType != string(ObjectBlob) {
		cancel()
		return nil, errors.InvalidArgument(
			"cat-file returned object type '%s' but expected '%s'", objectType, ObjectBlob)
	}

	contentSize := objectSize
	if sizeLimit > 0 && sizeLimit < contentSize {
		contentSize = sizeLimit
	}

	return &types.BlobReader{
		SHA:         sha,
		Size:        objectSize,
		ContentSize: contentSize,
		Content:     newLimitReaderCloser(stdOut, contentSize, cancel),
	}, nil
}

func newLimitReaderCloser(reader io.Reader, limit int64, stop func()) limitReaderCloser {
	return limitReaderCloser{
		reader: io.LimitReader(reader, limit),
		stop:   stop,
	}
}

type limitReaderCloser struct {
	reader io.Reader
	stop   func()
}

func (l limitReaderCloser) Read(p []byte) (n int, err error) {
	return l.reader.Read(p)
}

func (l limitReaderCloser) Close() error {
	l.stop()
	return nil
}
