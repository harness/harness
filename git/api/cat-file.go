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

package api

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
)

// WriteCloserError wraps an io.WriteCloser with an additional CloseWithError function.
type WriteCloserError interface {
	io.WriteCloser
	CloseWithError(err error) error
}

// CatFileBatch opens git cat-file --batch in the provided repo and returns a stdin pipe,
// a stdout reader and cancel function.
func CatFileBatch(
	ctx context.Context,
	repoPath string,
	alternateObjectDirs []string,
	flags ...command.CmdOptionFunc,
) (WriteCloserError, *bufio.Reader, func()) {
	const bufferSize = 32 * 1024
	// We often want to feed the commits in order into cat-file --batch,
	// followed by their trees and sub trees as necessary.
	batchStdinReader, batchStdinWriter := io.Pipe()
	batchStdoutReader, batchStdoutWriter := nio.Pipe(buffer.New(bufferSize))
	ctx, ctxCancel := context.WithCancel(ctx)
	closed := make(chan struct{})
	cancel := func() {
		ctxCancel()
		_ = batchStdinWriter.Close()
		_ = batchStdoutReader.Close()
		<-closed
	}

	// Ensure cancel is called as soon as the provided context is cancelled
	go func() {
		<-ctx.Done()
		cancel()
	}()

	go func() {
		stderr := bytes.Buffer{}
		cmd := command.New("cat-file",
			command.WithFlag("--batch"),
			command.WithAlternateObjectDirs(alternateObjectDirs...),
		)
		cmd.Add(flags...)
		err := cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdin(batchStdinReader),
			command.WithStdout(batchStdoutWriter),
			command.WithStderr(&stderr),
		)
		if err != nil {
			_ = batchStdoutWriter.CloseWithError(command.NewError(err, stderr.Bytes()))
			_ = batchStdinReader.CloseWithError(command.NewError(err, stderr.Bytes()))
		} else {
			_ = batchStdoutWriter.Close()
			_ = batchStdinReader.Close()
		}
		close(closed)
	}()

	// For simplicities sake we'll us a buffered reader to read from the cat-file --batch
	batchReader := bufio.NewReaderSize(batchStdoutReader, bufferSize)

	return batchStdinWriter, batchReader, cancel
}

type BatchHeaderResponse struct {
	SHA  sha.SHA
	Type string
	Size int64
}

// ReadBatchHeaderLine reads the header line from cat-file --batch
// <sha> SP <type> SP <size> LF
// sha is a 40byte not 20byte here.
func ReadBatchHeaderLine(rd *bufio.Reader) (*BatchHeaderResponse, error) {
	line, err := rd.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) == 1 {
		line, err = rd.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}
	idx := strings.IndexByte(line, ' ')
	if idx < 0 {
		return nil, errors.NotFound("missing space char for: %s", line)
	}
	id := line[:idx]
	objType := line[idx+1:]

	idx = strings.IndexByte(objType, ' ')
	if idx < 0 {
		return nil, errors.NotFound("sha '%s' not found", id)
	}

	sizeStr := objType[idx+1 : len(objType)-1]
	objType = objType[:idx]

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return &BatchHeaderResponse{
		SHA:  sha.Must(id),
		Type: objType,
		Size: size,
	}, nil
}
