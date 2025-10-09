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

package git

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
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
		return errors.InvalidArgument("git ref needs to be provided")
	}

	if params.Path == "" {
		return errors.InvalidArgument("file path needs to be provided")
	}

	if params.LineFrom < 0 || params.LineTo < 0 {
		return errors.InvalidArgument("line from and line to can't be negative")
	}

	if params.LineTo > 0 && params.LineFrom > params.LineTo {
		return errors.InvalidArgument("line from can't be after line after")
	}

	return nil
}

type BlamePart struct {
	Commit   *Commit            `json:"commit"`
	Lines    []string           `json:"lines"`
	Previous *BlamePartPrevious `json:"previous,omitempty"`
}

type BlamePartPrevious struct {
	CommitSHA sha.SHA `json:"commit_sha"`
	FileName  string  `json:"file_name"`
}

// Blame processes and streams the git blame output data.
// The function returns two channels: The data channel and the error channel.
// If any error happens during the operation it will be put to the error channel
// and the streaming will stop. Maximum of one error can be put on the channel.
func (s *Service) Blame(ctx context.Context, params *BlameParams) (<-chan *BlamePart, <-chan error) {
	ch := make(chan *BlamePart)
	chErr := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(chErr)

		if err := params.Validate(); err != nil {
			chErr <- err
			return
		}

		repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

		reader := s.git.Blame(ctx,
			repoPath, params.GitRef, params.Path,
			params.LineFrom, params.LineTo)

		for {
			part, errRead := reader.NextPart()
			if part == nil {
				return
			}

			commit, err := mapCommit(part.Commit)
			if err != nil {
				chErr <- fmt.Errorf("failed to map rpc commit: %w", err)
				return
			}

			lines := make([]string, len(part.Lines))
			copy(lines, part.Lines)

			next := &BlamePart{
				Commit: commit,
				Lines:  lines,
			}
			if part.Previous != nil {
				next.Previous = &BlamePartPrevious{
					CommitSHA: part.Previous.CommitSHA,
					FileName:  part.Previous.FileName,
				}
			}

			ch <- next

			if errRead != nil && errors.Is(errRead, io.EOF) {
				return
			}
		}
	}()

	return ch, chErr
}
