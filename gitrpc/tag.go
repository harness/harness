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
	"github.com/rs/zerolog/log"
)

type TagSortOption int

const (
	TagSortOptionDefault TagSortOption = iota
	TagSortOptionName
	TagSortOptionDate
)

type ListCommitTagsParams struct {
	// RepoUID is the uid of the git repository
	RepoUID       string
	IncludeCommit bool
	Query         string
	Sort          TagSortOption
	Order         SortOrder
	Page          int32
	PageSize      int32
}

type ListCommitTagsOutput struct {
	Tags []CommitTag
}

type CommitTag struct {
	Name        string
	SHA         string
	IsAnnotated bool
	Title       string
	Message     string
	Tagger      *Signature
	Commit      *Commit
}

func (c *Client) ListCommitTags(ctx context.Context, params *ListCommitTagsParams) (*ListCommitTagsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	stream, err := c.refService.ListCommitTags(ctx, &rpc.ListCommitTagsRequest{
		RepoUid:       params.RepoUID,
		IncludeCommit: params.IncludeCommit,
		Query:         params.Query,
		Sort:          mapToRPCListCommitTagsSortOption(params.Sort),
		Order:         mapToRPCSortOrder(params.Order),
		Page:          params.Page,
		PageSize:      params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for tags: %w", err)
	}

	// NOTE: don't use PageSize as initial slice capacity - as that theoretically could be MaxInt
	output := &ListCommitTagsOutput{
		Tags: make([]CommitTag, 0, 16),
	}
	for {
		var next *rpc.ListCommitTagsResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, processRPCErrorf(err, "received unexpected error from server")
		}
		if next.GetTag() == nil {
			return nil, fmt.Errorf("expected tag message")
		}

		var tag *CommitTag
		tag, err = mapRPCCommitTag(next.GetTag())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc tag: %w", err)
		}

		output.Tags = append(output.Tags, *tag)
	}

	err = stream.CloseSend()
	if err != nil {
		return nil, fmt.Errorf("failed to close stream")
	}

	return output, nil
}
