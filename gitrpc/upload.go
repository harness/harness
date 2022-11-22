// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
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
	file File,
	chunkSize int,
	send func(*rpc.FileUpload) error,
) error {
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
