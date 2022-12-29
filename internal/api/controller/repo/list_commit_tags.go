// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CommitTag struct {
	Name        string     `json:"name"`
	SHA         string     `json:"sha"`
	IsAnnotated bool       `json:"is_annotated"`
	Title       string     `json:"title,omitempty"`
	Message     string     `json:"message,omitempty"`
	Tagger      *Signature `json:"tagger,omitempty"`
	Commit      *Commit    `json:"commit,omitempty"`
}

/*
* ListCommitTags lists the commit tags of a repo.
 */
func (c *Controller) ListCommitTags(ctx context.Context, session *auth.Session,
	repoRef string, includeCommit bool, filter *types.TagFilter) ([]CommitTag, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, err
	}

	rpcOut, err := c.gitRPCClient.ListCommitTags(ctx, &gitrpc.ListCommitTagsParams{
		RepoUID:       repo.GitUID,
		IncludeCommit: includeCommit,
		Query:         filter.Query,
		Sort:          mapToRPCTagSortOption(filter.Sort),
		Order:         mapToRPCSortOrder(filter.Order),
		Page:          int32(filter.Page),
		PageSize:      int32(filter.Size),
	})
	if err != nil {
		return nil, err
	}

	tags := make([]CommitTag, len(rpcOut.Tags))
	for i := range rpcOut.Tags {
		tags[i], err = mapCommitTag(rpcOut.Tags[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map CommitTag: %w", err)
		}
	}

	return tags, nil
}

func mapToRPCTagSortOption(o enum.TagSortOption) gitrpc.TagSortOption {
	switch o {
	case enum.TagSortOptionDate:
		return gitrpc.TagSortOptionDate
	case enum.TagSortOptionName:
		return gitrpc.TagSortOptionName
	case enum.TagSortOptionDefault:
		return gitrpc.TagSortOptionDefault
	default:
		// no need to error out - just use default for sorting
		return gitrpc.TagSortOptionDefault
	}
}

func mapCommitTag(t gitrpc.CommitTag) (CommitTag, error) {
	var commit *Commit
	if t.Commit != nil {
		var err error
		commit, err = mapCommit(t.Commit)
		if err != nil {
			return CommitTag{}, err
		}
	}

	var tagger *Signature
	if t.Tagger != nil {
		var err error
		tagger, err = mapSignature(t.Tagger)
		if err != nil {
			return CommitTag{}, err
		}
	}

	return CommitTag{
		Name:        t.Name,
		SHA:         t.SHA,
		IsAnnotated: t.IsAnnotated,
		Title:       t.Title,
		Message:     t.Message,
		Tagger:      tagger,
		Commit:      commit,
	}, nil
}
