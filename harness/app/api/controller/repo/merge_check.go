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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"
)

type MergeCheck struct {
	Mergeable     bool     `json:"mergeable"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
}

func (c *Controller) MergeCheck(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	diffPath string,
) (MergeCheck, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return MergeCheck{}, err
	}

	dotRange, err := parseDotRangePath(diffPath)
	if err != nil {
		return MergeCheck{}, err
	}

	err = c.fetchDotRangeObjectsFromUpstream(ctx, session, repo, &dotRange)
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	readParams := git.CreateReadParams(repo)

	baseResult, err := c.git.ResolveRevision(ctx, git.ResolveRevisionParams{
		ReadParams: readParams,
		Revision:   dotRange.BaseRef,
	})
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to resolve base revision %s: %w", dotRange.BaseRef, err)
	}

	headResult, err := c.git.ResolveRevision(ctx, git.ResolveRevisionParams{
		ReadParams: readParams,
		Revision:   dotRange.HeadRef,
	})
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to resolve head revision %s: %w", dotRange.HeadRef, err)
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return MergeCheck{}, fmt.Errorf("failed to create rpc write params: %w", err)
	}

	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams: writeParams,
		BaseSHA:     baseResult.SHA,
		HeadSHA:     headResult.SHA,
	})
	if err != nil {
		// git.Merge works with commits and error is not user-friendly
		// and here we are modify base-ref and head-ref with user input
		// values.
		if uErr := api.AsUnrelatedHistoriesError(err); uErr != nil {
			uErr.BaseRef = dotRange.BaseRef
			uErr.HeadRef = dotRange.HeadRef
			return MergeCheck{}, uErr
		}
		return MergeCheck{}, fmt.Errorf("merge check execution failed: %w", err)
	}
	if len(mergeOutput.ConflictFiles) > 0 {
		return MergeCheck{
			Mergeable:     false,
			ConflictFiles: mergeOutput.ConflictFiles,
		}, nil
	}

	c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypeBranchMergableUpdated, mergeOutput)

	return MergeCheck{
		Mergeable: true,
	}, nil
}
