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
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type FileViewAddInput struct {
	Path      string `json:"path"`
	CommitSHA string `json:"commit_sha"`
}

func (f *FileViewAddInput) Validate() error {
	if f.Path == "" {
		return usererror.BadRequest("Path can't be empty")
	}
	if !git.ValidateCommitSHA(f.CommitSHA) {
		return usererror.BadRequest("Commit SHA is invalid")
	}

	return nil
}

// FileViewAdd marks a file as viewed.
// NOTE:
// We take the commit SHA from the user to ensure we mark as viewed only what the user actually sees.
// The downside is that the caller could provide a SHA that never was part of the PR in the first place.
// We can't block against that with our current data, as the existence of force push makes it impossible to verify
// whether the commit ever was part of the PR - it would require us to store the full pr.SourceSHA history.
//
//nolint:gocognit // refactor if needed.
func (c *Controller) FileViewAdd(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *FileViewAddInput,
) (*types.PullReqFileView, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	// retrieve file from both provided SHA and mergeBaseSHA to validate user input

	inNode, err := c.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          git.CreateReadParams(repo),
		GitREF:              in.CommitSHA,
		Path:                in.Path,
		IncludeLatestCommit: false,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf(
			"failed to get tree node '%s' for provided sha '%s': %w",
			in.Path,
			in.CommitSHA,
			err,
		)
	}

	// ensure provided path actually points to a blob or commit (submodule)
	if inNode != nil &&
		inNode.Node.Type != git.TreeNodeTypeBlob &&
		inNode.Node.Type != git.TreeNodeTypeCommit {
		return nil, usererror.BadRequestf("Provided path '%s' doesn't point to a file.", in.Path)
	}

	mergeBaseNode, err := c.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          git.CreateReadParams(repo),
		GitREF:              pr.MergeBaseSHA,
		Path:                in.Path,
		IncludeLatestCommit: false,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf(
			"failed to get tree node '%s' for MergeBaseSHA '%s': %w",
			in.Path,
			pr.MergeBaseSHA,
			err,
		)
	}

	// fail the call in case the file doesn't exist in either, or in case it didn't change.
	// NOTE: There is a RARE chance if the user provides an old SHA AND there's a new mergeBaseSHA
	// which now already contains the changes, that we return an error saying there are no changes
	// (even though with the old merge base there were). Effectively, it would lead to the users
	// 'file viewed' resetting when the user refreshes the page - we are okay with that.
	if inNode == nil && mergeBaseNode == nil {
		return nil, usererror.BadRequestf(
			"File '%s' neither found for merge-base '%s' nor for provided sha '%s'.",
			in.Path,
			pr.MergeBaseSHA,
			in.CommitSHA,
		)
	}
	if inNode != nil && mergeBaseNode != nil && inNode.Node.SHA == mergeBaseNode.Node.SHA {
		return nil, usererror.BadRequestf(
			"File '%s' is not part of changes between merge-base '%s' and provided sha '%s'.",
			in.Path,
			pr.MergeBaseSHA,
			in.CommitSHA,
		)
	}

	// in case of deleted file set sha to nilsha - that's how git diff treats it, too.
	sha := types.NilSHA
	if inNode != nil {
		sha = inNode.Node.SHA
	}

	now := time.Now().UnixMilli()
	fileView := &types.PullReqFileView{
		PullReqID:   pr.ID,
		PrincipalID: session.Principal.ID,

		Path: in.Path,
		SHA:  sha,

		// always add as non-obsolete, even if the file view is derived from a non-latest commit sha.
		// The file sha ensures that the user's review is out of date in case the file changed in the meanwhile.
		// And in the rare case of the file having changed and changed back between the two commits,
		// the content the reviewer just approved on is the same, so it's actually good to not mark it as obsolete.
		Obsolete: false,

		Created: now,
		Updated: now,
	}

	err = c.fileViewStore.Upsert(ctx, fileView)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert file view information in db: %w", err)
	}

	return fileView, nil
}
