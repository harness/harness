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
	"context"
	"errors"
	"io"

	"github.com/harness/gitness/gitrpc/rpc"
)

type BlameParams struct {
	ReadParams
	GitRef string
	Path   string

	// LineFrom allows to restrict the blame output to only lines starting from the provided line number (inclusive).
	// Optional, ignored if value is 0.
	LineFrom int

	// LineTo allows to restrict the blame output to only lines up to the provided line number (inclusive).
	// Optional, ignored if value is 0.
	LineTo int
}

func (params *BlameParams) Validate() error {
	if params == nil {
		return ErrNoParamsProvided
	}

	if err := params.ReadParams.Validate(); err != nil {
		return err
	}

	if params.GitRef == "" {
		return Errorf(StatusInvalidArgument, "git ref needs to be provided")
	}

	if params.Path == "" {
		return Errorf(StatusInvalidArgument, "file path needs to be provided")
	}

	if params.LineFrom < 0 || params.LineTo < 0 {
		return Errorf(StatusInvalidArgument, "line from and line to can't be negative")
	}

	if params.LineTo > 0 && params.LineFrom > params.LineTo {
		return Errorf(StatusInvalidArgument, "line from can't be after line after")
	}

	return nil
}

type BlamePart struct {
	Commit *Commit  `json:"commit"`
	Lines  []string `json:"lines"`
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

			lines := make([]string, len(part.Lines))
			for i, line := range part.Lines {
				lines[i] = string(line)
			}

			ch <- &BlamePart{Commit: commit, Lines: lines}
		}
	}()

	return ch, chErr
}
