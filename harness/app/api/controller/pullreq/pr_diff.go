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

package pullreq

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	gittypes "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RawDiff writes raw git diff to writer w.
func (c *Controller) RawDiff(
	ctx context.Context,
	w io.Writer,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	setSHAs func(sourceSHA, mergeBaseSHA string),
	files ...gittypes.FileDiffRequest,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if setSHAs != nil {
		setSHAs(pr.SourceSHA, pr.MergeBaseSHA)
	}

	return c.git.RawDiff(ctx, w, &git.DiffParams{
		ReadParams: git.CreateReadParams(repo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
		MergeBase:  true,
	}, files...)
}

func (c *Controller) Diff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	setSHAs func(sourceSHA, mergeBaseSHA string),
	includePatch bool,
	ignoreWhitespace bool,
	files ...gittypes.FileDiffRequest,
) (types.Stream[*git.FileDiff], error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if setSHAs != nil {
		setSHAs(pr.SourceSHA, pr.MergeBaseSHA)
	}

	reader := git.NewStreamReader(c.git.Diff(ctx, &git.DiffParams{
		ReadParams:       git.CreateReadParams(repo),
		BaseRef:          pr.MergeBaseSHA,
		HeadRef:          pr.SourceSHA,
		MergeBase:        true,
		IncludePatch:     includePatch,
		IgnoreWhitespace: ignoreWhitespace,
	}, files...))

	return reader, nil
}
