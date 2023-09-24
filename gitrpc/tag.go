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

type CreateCommitTagParams struct {
	WriteParams
	Name string

	// Target is the commit (or points to the commit) the new tag will be pointing to.
	Target string

	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated
	Message string

	// Tagger overwrites the git author used in case the tag is annotated
	// (optional, default: actor)
	Tagger *Identity
	// TaggerDate overwrites the git author date used in case the tag is annotated
	// (optional, default: current time on server)
	TaggerDate *time.Time
}

func (p *CreateCommitTagParams) Validate() error {
	if p == nil {
		return ErrNoParamsProvided
	}

	if p.Name == "" {
		return errors.New("tag name cannot be empty")
	}
	if p.Target == "" {
		return errors.New("target cannot be empty")
	}

	return nil
}

type CreateCommitTagOutput struct {
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
func (c *Client) CreateCommitTag(ctx context.Context, params *CreateCommitTagParams) (*CreateCommitTagOutput, error) {
	err := params.Validate()

	if err != nil {
		return nil, err
	}

	resp, err := c.refService.CreateCommitTag(ctx, &rpc.CreateCommitTagRequest{
		Base:       mapToRPCWriteRequest(params.WriteParams),
		Target:     params.Target,
		TagName:    params.Name,
		Message:    params.Message,
		Tagger:     mapToRPCIdentityOptional(params.Tagger),
		TaggerDate: mapToRPCTimeOptional(params.TaggerDate),
	})

	if err != nil {
		return nil, processRPCErrorf(err, "Failed to create tag %s", params.Name)
	}

	var commitTag *CommitTag
	commitTag, err = mapRPCCommitTag(resp.GetTag())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc tag: %w", err)
	}

	return &CreateCommitTagOutput{
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
