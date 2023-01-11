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

type GetCommitParams struct {
	ReadParams
	// SHA is the git commit sha
	SHA string
}

type GetCommitOutput struct {
	Commit Commit
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

func (c *Client) GetCommit(ctx context.Context, params *GetCommitParams) (*GetCommitOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	result, err := c.repoService.GetCommit(ctx, &rpc.GetCommitRequest{
		Base: mapToRPCReadRequest(params.ReadParams),
		Sha:  params.SHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	commit, err := mapRPCCommit(result.GetCommit())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc commit: %w", err)
	}

	return &GetCommitOutput{
		Commit: *commit,
	}, nil
}

type ListCommitsParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF string
	// After is a git reference (branch / tag / commit SHA)
	// If provided, commits only up to that reference will be returned (exlusive)
	After string
	Page  int32
	Limit int32
}

type ListCommitsOutput struct {
	Commits []Commit
}

func (c *Client) ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListCommits(ctx, &rpc.ListCommitsRequest{
		Base:   mapToRPCReadRequest(params.ReadParams),
		GitRef: params.GitREF,
		After:  params.After,
		Page:   params.Page,
		Limit:  params.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for commits: %w", err)
	}
	// NOTE: don't use PageSize as initial slice capacity - as that theoretically could be MaxInt
	output := &ListCommitsOutput{
		Commits: make([]Commit, 0, 16),
	}

	for {
		var next *rpc.ListCommitsResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, processRPCErrorf(err, "received unexpected error from server")
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

type GetCommitDivergencesParams struct {
	ReadParams
	MaxCount int32
	Requests []CommitDivergenceRequest
}

type GetCommitDivergencesOutput struct {
	Divergences []CommitDivergence
}

// CommitDivergenceRequest contains the refs for which the converging commits should be counted.
type CommitDivergenceRequest struct {
	// From is the ref from which the counting of the diverging commits starts.
	From string
	// To is the ref at which the counting of the diverging commits ends.
	To string
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32
}

func (c *Client) GetCommitDivergences(ctx context.Context,
	params *GetCommitDivergencesParams) (*GetCommitDivergencesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	// build rpc request
	req := &rpc.GetCommitDivergencesRequest{
		Base:     mapToRPCReadRequest(params.ReadParams),
		MaxCount: params.MaxCount,
		Requests: make([]*rpc.CommitDivergenceRequest, len(params.Requests)),
	}
	for i := range params.Requests {
		req.Requests[i] = &rpc.CommitDivergenceRequest{
			From: params.Requests[i].From,
			To:   params.Requests[i].To,
		}
	}
	resp, err := c.repoService.GetCommitDivergences(ctx, req)
	if err != nil {
		return nil, processRPCErrorf(err, "failed to get diverging commits from server")
	}

	divergences := resp.GetDivergences()
	if divergences == nil {
		return nil, fmt.Errorf("server response divergences were nil")
	}

	// build output
	output := &GetCommitDivergencesOutput{
		Divergences: make([]CommitDivergence, len(divergences)),
	}
	for i := range divergences {
		if divergences[i] == nil {
			return nil, fmt.Errorf("server returned nil divergence")
		}

		output.Divergences[i] = CommitDivergence{
			Ahead:  divergences[i].Ahead,
			Behind: divergences[i].Behind,
		}
	}

	return output, nil
}
