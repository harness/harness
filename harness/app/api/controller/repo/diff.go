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
	"io"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	gittypes "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) RawDiff(
	ctx context.Context,
	w io.Writer,
	session *auth.Session,
	repoRef string,
	path string,
	ignoreWhitespace bool,
	files ...gittypes.FileDiffRequest,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return err
	}

	dotRange, err := parseDotRangePath(path)
	if err != nil {
		return err
	}

	err = c.fetchDotRangeObjectsFromUpstream(ctx, session, repo, &dotRange)
	if err != nil {
		return fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	return c.git.RawDiff(ctx, w, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          dotRange.BaseRef,
		HeadRef:          dotRange.HeadRef,
		MergeBase:        dotRange.MergeBase,
		IgnoreWhitespace: ignoreWhitespace,
	}, files...)
}

func (c *Controller) CommitDiff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	rev string,
	ignoreWhitespace bool,
	w io.Writer,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return err
	}

	return c.git.CommitDiff(ctx, &git.GetCommitParams{
		ReadParams:       git.CreateReadParams(repo),
		Revision:         rev,
		IgnoreWhitespace: ignoreWhitespace,
	}, w)
}

func (c *Controller) DiffStats(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	path string,
	ignoreWhitespace bool,
) (types.DiffStats, error) {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return types.DiffStats{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView); err != nil {
		return types.DiffStats{}, err
	}

	dotRange, err := parseDotRangePath(path)
	if err != nil {
		return types.DiffStats{}, err
	}

	err = c.fetchDotRangeObjectsFromUpstream(ctx, session, repo, &dotRange)
	if err != nil {
		return types.DiffStats{}, fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	output, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          dotRange.BaseRef,
		HeadRef:          dotRange.HeadRef,
		MergeBase:        dotRange.MergeBase,
		IgnoreWhitespace: ignoreWhitespace,
	})
	if err != nil {
		return types.DiffStats{}, err
	}

	return types.NewDiffStats(output.Commits, output.FilesChanged, output.Additions, output.Deletions), nil
}

func (c *Controller) Diff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	path string,
	includePatch bool,
	ignoreWhitespace bool,
	files ...gittypes.FileDiffRequest,
) (types.Stream[*git.FileDiff], error) {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView); err != nil {
		return nil, err
	}

	dotRange, err := parseDotRangePath(path)
	if err != nil {
		return nil, err
	}

	err = c.fetchDotRangeObjectsFromUpstream(ctx, session, repo, &dotRange)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	reader := git.NewStreamReader(c.git.Diff(ctx, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          dotRange.BaseRef,
		HeadRef:          dotRange.HeadRef,
		MergeBase:        dotRange.MergeBase,
		IncludePatch:     includePatch,
		IgnoreWhitespace: ignoreWhitespace,
	}, files...))

	return reader, nil
}

func (c *Controller) fetchDotRangeObjectsFromUpstream(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	dotRange *DotRange,
) error {
	if dotRange.BaseUpstream {
		refSHA, _, err := c.fetchUpstreamRevision(ctx, session, repoForkCore, dotRange.BaseRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		dotRange.BaseUpstream = false
		dotRange.BaseRef = refSHA.String()
	}

	if dotRange.HeadUpstream {
		refSHA, _, err := c.fetchUpstreamRevision(ctx, session, repoForkCore, dotRange.HeadRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		dotRange.HeadUpstream = false
		dotRange.HeadRef = refSHA.String()
	}

	return nil
}
