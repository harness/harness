// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/rpc"

	"golang.org/x/sync/errgroup"
)

type DiffParams struct {
	ReadParams
	BaseRef   string
	HeadRef   string
	MergeBase bool
}

func (p DiffParams) Validate() error {
	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if p.HeadRef == "" {
		return errors.New("head ref cannot be empty")
	}
	return nil
}

func (c *Client) RawDiff(ctx context.Context, params *DiffParams, out io.Writer) error {
	if err := params.Validate(); err != nil {
		return err
	}
	diff, err := c.diffService.RawDiff(ctx, &rpc.DiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return err
	}

	reader := streamio.NewReader(func() ([]byte, error) {
		var resp *rpc.RawDiffResponse
		resp, err = diff.Recv()
		return resp.GetData(), err
	})

	if _, err = io.Copy(out, reader); err != nil {
		return fmt.Errorf("copy rpc data: %w", err)
	}

	return nil
}

type DiffShortStatOutput struct {
	Files     int
	Additions int
	Deletions int
}

// DiffShortStat returns files changed, additions and deletions metadata.
func (c *Client) DiffShortStat(ctx context.Context, params *DiffParams) (DiffShortStatOutput, error) {
	if err := params.Validate(); err != nil {
		return DiffShortStatOutput{}, err
	}
	stat, err := c.diffService.DiffShortStat(ctx, &rpc.DiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return DiffShortStatOutput{}, err
	}
	return DiffShortStatOutput{
		Files:     int(stat.GetFiles()),
		Additions: int(stat.GetAdditions()),
		Deletions: int(stat.GetDeletions()),
	}, nil
}

type DiffStatsOutput struct {
	Commits      int
	FilesChanged int
}

func (c *Client) DiffStats(ctx context.Context, params *DiffParams) (DiffStatsOutput, error) {
	// declare variables which will be used in go routines,
	// no need for atomic operations because writing and reading variable
	// doesn't happen at the same time
	var (
		totalCommits int
		totalFiles   int
	)

	errGroup, groupCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		// read total commits

		options := &GetCommitDivergencesParams{
			ReadParams: params.ReadParams,
			Requests: []CommitDivergenceRequest{
				{
					From: params.HeadRef,
					To:   params.BaseRef,
				},
			},
		}

		rpcOutput, err := c.GetCommitDivergences(groupCtx, options)
		if err != nil {
			return fmt.Errorf("failed to count pull request commits: %w", err)
		}
		if len(rpcOutput.Divergences) > 0 {
			totalCommits = int(rpcOutput.Divergences[0].Ahead)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read short stat
		stat, err := c.DiffShortStat(groupCtx, &DiffParams{
			ReadParams: params.ReadParams,
			BaseRef:    params.BaseRef,
			HeadRef:    params.HeadRef,
			MergeBase:  true, // must be true, because commitDivergences use tripple dot notation
		})
		if err != nil {
			return fmt.Errorf("failed to count pull request file changes: %w", err)
		}
		totalFiles = stat.Files
		return nil
	})

	err := errGroup.Wait()
	if err != nil {
		return DiffStatsOutput{}, err
	}

	return DiffStatsOutput{
		Commits:      totalCommits,
		FilesChanged: totalFiles,
	}, nil
}
