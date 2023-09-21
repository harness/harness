// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
)

const (
	// TODO: this should be configurable
	FileTransferChunkSize = 1024
)

type File struct {
	Path    string
	Content []byte
}

func uploadFile(
	ctx context.Context,
	file File,
	chunkSize int,
	send func(*rpc.FileUpload) error,
) error {
	log := log.Ctx(ctx)

	log.Info().Msgf("start sending %v", file.Path)

	// send filename message
	header := &rpc.FileUpload{
		Data: &rpc.FileUpload_Header{
			Header: &rpc.FileUploadHeader{
				Path: file.Path,
			},
		},
	}
	if err := send(header); err != nil {
		return fmt.Errorf("failed to send file upload header: %w", err)
	}

	err := sendChunks(file.Content, chunkSize, func(c *rpc.Chunk) error {
		return send(&rpc.FileUpload{
			Data: &rpc.FileUpload_Chunk{
				Chunk: c,
			},
		})
	})
	if err != nil {
		return fmt.Errorf("failed to send file data: %w", err)
	}

	log.Info().Msgf("completed sending %v", file.Path)

	return nil
}

func sendChunks(
	content []byte,
	chunkSize int,
	send func(*rpc.Chunk) error) error {
	buffer := make([]byte, chunkSize)
	reader := bytes.NewReader(content)

	for {
		n, err := reader.Read(buffer)
		if errors.Is(err, io.EOF) {
			err = send(&rpc.Chunk{
				Eof:  true,
				Data: buffer[:n],
			})
			if err != nil {
				return err
			}

			break
		}
		if err != nil {
			return fmt.Errorf("cannot read buffer: %w", err)
		}

		err = send(&rpc.Chunk{
			Eof:  false,
			Data: buffer[:n],
		})
		if err != nil {
			return err
		}
	}

	return nil
}
