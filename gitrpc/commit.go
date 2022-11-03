// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"
	"github.com/rs/zerolog/log"
)

type ListCommitsParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF   string
	Page     int32
	PageSize int32
}

type ListCommitsOutput struct {
	TotalCount int64
	Commits    []Commit
}

type Commit struct {
	SHA       string
	Title     string
	Message   string
	Author    Signature
	Committer Signature
}

type Signature struct {
	Identity Identity
	When     time.Time
}

type Identity struct {
	Name  string
	Email string
}

func (c *Client) ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListCommits(ctx, &rpc.ListCommitsRequest{
		RepoUid:  params.RepoUID,
		GitRef:   params.GitREF,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for commits: %w", err)
	}

	// get header first
	header, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("error occured while receiving header: %w", err)
	}
	if header.GetHeader() == nil {
		return nil, fmt.Errorf("header missing")
	}

	// NOTE: don't use PageSize as initial slice capacity - as that theoretically could be MaxInt
	output := &ListCommitsOutput{
		TotalCount: header.GetHeader().TotalCount,
		Commits:    make([]Commit, 0, 16),
	}

	for {
		var next *rpc.ListCommitsResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}
		if next.GetCommit() == nil {
			return nil, fmt.Errorf("expected commit message")
		}

		var commit *Commit
		commit, err = mapRPCCommit(next.GetCommit())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}

		output.Commits = append(output.Commits, *commit)
	}

	return output, nil
}
