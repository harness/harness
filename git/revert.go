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

package git

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/rs/zerolog/log"
)

// RevertParams is input structure object for the revert operation.
type RevertParams struct {
	WriteParams

	ParentCommitSHA sha.SHA
	FromCommitSHA   sha.SHA
	ToCommitSHA     sha.SHA

	RevertBranch string

	Message string

	Committer     *Identity
	CommitterDate *time.Time
	Author        *Identity
	AuthorDate    *time.Time
}

func (p *RevertParams) Validate() error {
	if err := p.WriteParams.Validate(); err != nil {
		return err
	}

	if p.Message == "" {
		return errors.InvalidArgument("commit message is empty")
	}

	if p.RevertBranch == "" {
		return errors.InvalidArgument("revert branch is missing")
	}

	return nil
}

type RevertOutput struct {
	CommitSHA sha.SHA
}

// Revert creates a revert commit. The revert commit contains all changes introduced
// by the commits between params.FromCommitSHA and params.ToCommitSHA.
// The newly created commit will have the parent set as params.ParentCommitSHA.
// This method can be used to revert a pull request:
// * params.ParentCommit = pr.MergeSHA
// * params.FromCommitSHA = pr.MergeBaseSHA
// * params.ToCommitSHA = pr.SourceSHA.
func (s *Service) Revert(ctx context.Context, params *RevertParams) (RevertOutput, error) {
	err := params.Validate()
	if err != nil {
		return RevertOutput{}, fmt.Errorf("params not valid: %w", err)
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	// Check the merge base commit of the FromCommit and the ToCommit.
	// We expect that the merge base would be equal to the FromCommit.
	// This guaranties that the FromCommit is an ancestor of the ToCommit
	// and that the two commits have simple commit history.
	// The diff could be found with the simple 'git diff',
	// rather than going though the merge base with 'git diff --merge-base'.

	if params.FromCommitSHA.Equal(params.ToCommitSHA) {
		return RevertOutput{}, errors.InvalidArgument("from and to commits can't be the same commit.")
	}

	mergeBaseCommitSHA, _, err := s.git.GetMergeBase(ctx, repoPath, "origin",
		params.FromCommitSHA.String(), params.ToCommitSHA.String(), false)
	if err != nil {
		return RevertOutput{}, fmt.Errorf("failed to get merge base: %w", err)
	}

	if !params.FromCommitSHA.Equal(mergeBaseCommitSHA) {
		return RevertOutput{}, errors.InvalidArgument("from and to commits must not branch out.")
	}

	// Set the author and the committer. The rules for setting these are the same as for the Merge method.

	now := time.Now().UTC()

	committer := api.Signature{Identity: api.Identity(params.Actor), When: now}

	if params.Committer != nil {
		committer.Identity = api.Identity(*params.Committer)
	}
	if params.CommitterDate != nil {
		committer.When = *params.CommitterDate
	}

	author := committer

	if params.Author != nil {
		author.Identity = api.Identity(*params.Author)
	}
	if params.AuthorDate != nil {
		author.When = *params.AuthorDate
	}

	// Prepare the message for the revert commit.

	message := parser.CleanUpWhitespace(params.Message)

	//  Reference updater

	refRevertBranch, err := GetRefPath(params.RevertBranch, gitenum.RefTypeBranch)
	if err != nil {
		return RevertOutput{}, fmt.Errorf("failed to generate revert branch ref name: %w", err)
	}

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return RevertOutput{}, fmt.Errorf("failed to create reference updater: %w", err)
	}

	// Temp file to hold the diff patch.

	diffPatch, err := os.CreateTemp(s.sharedRepoRoot, "revert-*.patch")
	if err != nil {
		return RevertOutput{}, fmt.Errorf("failed to create temporary file to hold the diff patch: %w", err)
	}

	diffPatchName := diffPatch.Name()

	defer func() {
		if err = os.Remove(diffPatchName); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("path", diffPatchName).Msg("failed to remove temp file")
		}
	}()

	// Create the revert commit

	var commitSHA sha.SHA

	err = sharedrepo.Run(ctx, refUpdater, s.sharedRepoRoot, repoPath, func(s *sharedrepo.SharedRepo) error {
		if err := s.WriteDiff(ctx, params.ToCommitSHA.String(), params.FromCommitSHA.String(), diffPatch); err != nil {
			return fmt.Errorf("failed to find diff between the two commits: %w", err)
		}

		if err := diffPatch.Close(); err != nil {
			return fmt.Errorf("failed to close patch file: %w", err)
		}

		fInfo, err := os.Lstat(diffPatchName)
		if err != nil {
			return fmt.Errorf("failed to lstat diff path file: %w", err)
		}

		if fInfo.Size() == 0 {
			return errors.InvalidArgument("can't revert an empty diff")
		}

		if err := s.SetIndex(ctx, params.ParentCommitSHA); err != nil {
			return fmt.Errorf("failed to set parent commit index: %w", err)
		}

		if err := s.ApplyToIndex(ctx, diffPatchName); err != nil {
			return fmt.Errorf("failed to apply revert diff: %w", err)
		}

		treeSHA, err := s.WriteTree(ctx)
		if err != nil {
			return fmt.Errorf("failed to write revert tree: %w", err)
		}

		commitSHA, err = s.CommitTree(ctx, &author, &committer, treeSHA, message, false, params.ParentCommitSHA)
		if err != nil {
			return fmt.Errorf("failed to create revert commit: %w", err)
		}

		refUpdates := []hook.ReferenceUpdate{
			{
				Ref: refRevertBranch,
				Old: sha.Nil, // Expect that the revert branch doesn't exist.
				New: commitSHA,
			},
		}

		err = refUpdater.Init(ctx, refUpdates)
		if err != nil {
			return fmt.Errorf("failed to set init value of the revert reference %s: %w", refRevertBranch, err)
		}

		return nil
	})
	if err != nil {
		return RevertOutput{}, fmt.Errorf("failed to revert: %w", err)
	}

	return RevertOutput{
		CommitSHA: commitSHA,
	}, nil
}
