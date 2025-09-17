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
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
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

	info, err := parseDiffPath(path)
	if err != nil {
		return err
	}

	err = c.fetchDiffUpstreamRef(ctx, session, repo, &info)
	if err != nil {
		return fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	return c.git.RawDiff(ctx, w, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          info.BaseRef,
		HeadRef:          info.HeadRef,
		MergeBase:        info.MergeBase,
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

type CompareInfo struct {
	BaseRef      string
	BaseUpstream bool
	HeadRef      string
	HeadUpstream bool
	MergeBase    bool
}

func parseDiffPath(path string) (CompareInfo, error) {
	mergeBase := true
	infos := strings.SplitN(path, "...", 2)
	if len(infos) != 2 {
		mergeBase = false
		infos = strings.SplitN(path, "..", 2)
		if len(infos) != 2 {
			return CompareInfo{}, usererror.BadRequestf("Invalid format %q", path)
		}
	}

	info := CompareInfo{
		BaseRef:   infos[0],
		HeadRef:   infos[1],
		MergeBase: mergeBase,
	}

	const upstreamMarker = "upstream:"

	info.BaseRef, info.BaseUpstream = strings.CutPrefix(info.BaseRef, upstreamMarker)
	info.HeadRef, info.HeadUpstream = strings.CutPrefix(info.HeadRef, upstreamMarker)

	if info.BaseUpstream && info.HeadUpstream {
		return CompareInfo{}, usererror.BadRequestf("Only one upstream reference is allowed: %q", path)
	}

	return info, nil
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

	info, err := parseDiffPath(path)
	if err != nil {
		return types.DiffStats{}, err
	}

	err = c.fetchDiffUpstreamRef(ctx, session, repo, &info)
	if err != nil {
		return types.DiffStats{}, fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	output, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          info.BaseRef,
		HeadRef:          info.HeadRef,
		MergeBase:        info.MergeBase,
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

	info, err := parseDiffPath(path)
	if err != nil {
		return nil, err
	}

	err = c.fetchDiffUpstreamRef(ctx, session, repo, &info)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch diff upstream ref: %w", err)
	}

	reader := git.NewStreamReader(c.git.Diff(ctx, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          info.BaseRef,
		HeadRef:          info.HeadRef,
		MergeBase:        info.MergeBase,
		IncludePatch:     includePatch,
		IgnoreWhitespace: ignoreWhitespace,
	}, files...))

	return reader, nil
}

func (c *Controller) fetchDiffUpstreamRef(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	compareInfo *CompareInfo,
) error {
	if compareInfo.BaseUpstream {
		refSHA, _, err := c.fetchUpstreamRevision(ctx, session, repoForkCore, compareInfo.BaseRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		compareInfo.BaseUpstream = false
		compareInfo.BaseRef = refSHA.String()
	}

	if compareInfo.HeadUpstream {
		refSHA, _, err := c.fetchUpstreamRevision(ctx, session, repoForkCore, compareInfo.HeadRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		compareInfo.HeadUpstream = false
		compareInfo.HeadRef = refSHA.String()
	}

	return nil
}
