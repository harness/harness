// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/rpc"
)

type BlameParams struct {
	ReadParams
	GitRef   string
	Path     string
	LineFrom int
	LineTo   int
}

func (params *BlameParams) Validate() error {
	if params == nil {
		return ErrNoParamsProvided
	}

	if err := params.ReadParams.Validate(); err != nil {
		return err
	}

	if params.GitRef == "" {
		return fmt.Errorf("git ref needs to be provided: %w", ErrInvalidArgument)
	}

	if params.Path == "" {
		return fmt.Errorf("file path needs to be provided: %w", ErrInvalidArgument)
	}

	if params.LineFrom < 0 || params.LineTo < 0 {
		return fmt.Errorf("line from and line to can't be negative: %w", ErrInvalidArgument)
	}

	if params.LineTo > 0 && params.LineFrom > params.LineTo {
		return fmt.Errorf("line from can't be after line after: %w", ErrInvalidArgument)
	}

	return nil
}

type BlamePart struct {
	Commit *Commit
	Lines  []string
}

// Blame processes and streams the git blame output data.
// The function returns two channels: The data channel and the error channel.
// If any error happens during the operation it will be put to the error channel
// and the streaming will stop. Maximum of one error can be put on the channel.
func (c *Client) Blame(ctx context.Context, params *BlameParams) (<-chan *BlamePart, <-chan error) {
	ch := make(chan *BlamePart)
	chErr := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(chErr)

		if err := params.Validate(); err != nil {
			chErr <- err
			return
		}

		stream, err := c.blameService.Blame(ctx, &rpc.BlameRequest{
			Base:   mapToRPCReadRequest(params.ReadParams),
			GitRef: params.GitRef,
			Path:   params.Path,
			Range: &rpc.LineRange{
				From: int32(params.LineFrom),
				To:   int32(params.LineTo),
			},
		})
		if err != nil {
			chErr <- processRPCErrorf(err, "failed to get blame info from server")
			return
		}

		for {
			var part *rpc.BlamePart

			part, err = stream.Recv()
			if err != nil && !errors.Is(err, io.EOF) {
				chErr <- processRPCErrorf(err, "blame failed")
				return
			}

			if part == nil {
				return
			}

			var commit *Commit

			commit, err = mapRPCCommit(part.Commit)
			if err != nil {
				chErr <- processRPCErrorf(err, "failed to map rpc commit")
				return
			}

			ch <- &BlamePart{Commit: commit, Lines: part.Lines}
		}
	}()

	return ch, chErr
}
