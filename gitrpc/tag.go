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
	ReadParams
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

type CreateTagParams struct {
	WriteParams
	Name    string
	SHA     string
	Message string
}

func (p *CreateTagParams) Validate() error {
	if p == nil {
		return ErrNoParamsProvided
	}

	if p.Name == "" {
		return errors.New("Tag name cannot be empty")
	}
	if p.SHA == "" {
		return errors.New("Target cannot be empty")
	}
	if p.Message == "" {
		return errors.New("Message cannot be empty")
	}
	return nil
}

type CreateTagOutput struct {
	CommitTag
}

type DeleteTagParams struct {
	WriteParams
	Name string
}

func (p DeleteTagParams) Validate() error {
	if p.Name == "" {
		return errors.New("tag name cannot be empty")
	}
	return nil
}

func (c *Client) ListCommitTags(ctx context.Context, params *ListCommitTagsParams) (*ListCommitTagsOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	stream, err := c.refService.ListCommitTags(ctx, &rpc.ListCommitTagsRequest{
		Base:          mapToRPCReadRequest(params.ReadParams),
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
func (c *Client) CreateTag(ctx context.Context, params *CreateTagParams) (*CreateTagOutput, error) {

	err := params.Validate()

	if err != nil {
		return nil, err
	}

	resp, err := c.refService.CreateTag(ctx, &rpc.CreateTagRequest{
		Base:    mapToRPCWriteRequest(params.WriteParams),
		Sha:     params.SHA,
		TagName: params.Name,
		Message: params.Message,
	})

	if err != nil {
		return nil, processRPCErrorf(err, "Failed to create tag %s", params.Name)
	}

	var commitTag *CommitTag
	commitTag, err = mapRPCCommitTag(resp.GetTag())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc tag: %w", err)
	}

	return &CreateTagOutput{
		CommitTag: *commitTag,
	}, nil

}

func (c *Client) DeleteTag(ctx context.Context, params *DeleteTagParams) error {

	err := params.Validate()

	if err != nil {
		return err
	}

	_, err = c.refService.DeleteTag(ctx, &rpc.DeleteTagRequest{
		Base:    mapToRPCWriteRequest(params.WriteParams),
		TagName: params.Name,
	})

	if err != nil {
		return processRPCErrorf(err, "Failed to create tag %s", params.Name)
	}
	return nil
}
