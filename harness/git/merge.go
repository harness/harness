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
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/merge"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"
)

// MergeParams is input structure object for merging operation.
type MergeParams struct {
	WriteParams

	// BaseSHA is the target SHA when we want to merge. Either BaseSHA or BaseBranch must be provided.
	BaseSHA sha.SHA
	// BaseBranch is the target branch where we want to merge. Either BaseSHA or BaseBranch must be provided.
	BaseBranch string

	// HeadSHA is the source commit we want to merge onto the base. Either HeadSHA or HeadBranch must be provided.
	HeadSHA sha.SHA
	// HeadBranch is the source branch we want to merge. Either HeadSHA or HeadBranch must be provided.
	HeadBranch string

	// HeadBranchExpectedSHA is commit SHA on the HeadBranch. Ignored if HeadSHA is provided instead.
	// If HeadBranchExpectedSHA is older than the HeadBranch latest SHA then merge will fail.
	HeadBranchExpectedSHA sha.SHA

	// Merge is the message of the commit that would be created. Ignored for Rebase and FastForward.
	Message string

	// Committer overwrites the git committer used for committing the files
	// (optional, default: actor)
	Committer *Identity
	// CommitterDate overwrites the git committer date used for committing the files
	// (optional, default: current time on server)
	CommitterDate *time.Time
	// Author overwrites the git author used for committing the files
	// (optional, default: committer)
	Author *Identity
	// AuthorDate overwrites the git author date used for committing the files
	// (optional, default: committer date)
	AuthorDate *time.Time

	Refs []RefUpdate

	Force            bool
	DeleteHeadBranch bool

	Method enum.MergeMethod
}

type RefUpdate struct {
	// Name is the full name of the reference.
	Name string

	// Old is the expected current value of the reference.
	// If it's empty, the old value of the reference can be any value.
	Old sha.SHA

	// New is the desired value for the reference.
	// If it's empty, the reference would be set to the resulting commit SHA of the merge.
	New sha.SHA
}

func (p *MergeParams) Validate() error {
	if err := p.WriteParams.Validate(); err != nil {
		return err
	}

	if p.BaseBranch == "" && p.BaseSHA.IsEmpty() {
		return errors.InvalidArgument("either base branch or commit SHA is mandatory")
	}

	if p.HeadBranch == "" && p.HeadSHA.IsEmpty() {
		return errors.InvalidArgument("either head branch or head SHA is mandatory")
	}

	for _, ref := range p.Refs {
		if ref.Name == "" {
			return errors.InvalidArgument("ref name has to be provided")
		}
	}

	return nil
}

// MergeOutput is result object from merging and returns
// base, head and commit sha.
type MergeOutput struct {
	// BaseSHA is the sha of the latest commit on the base branch that was used for merging.
	BaseSHA sha.SHA
	// HeadSHA is the sha of the latest commit on the head branch that was used for merging.
	HeadSHA sha.SHA
	// MergeBaseSHA is the sha of the merge base of the HeadSHA and BaseSHA
	MergeBaseSHA sha.SHA
	// MergeSHA is the sha of the commit after merging HeadSHA with BaseSHA.
	MergeSHA sha.SHA

	CommitCount      int
	ChangedFileCount int
	Additions        int
	Deletions        int
	ConflictFiles    []string
}

// Merge method executes git merge operation. Refs can be sha, branch or tag.
// Based on input params.RefType merge can do checking or final merging of two refs.
// some examples:
//
//	params.RefType = Undefined -> discard merge commit (only performs a merge check).
//	params.RefType = Raw and params.RefName = refs/pull/1/ref will push to refs/pullreq/1/ref
//	params.RefType = RefTypeBranch and params.RefName = "somebranch" -> merge and push to refs/heads/somebranch
//	params.RefType = RefTypePullReqHead and params.RefName = "1" -> merge and push to refs/pullreq/1/head
//	params.RefType = RefTypePullReqMerge and params.RefName = "1" -> merge and push to refs/pullreq/1/merge
//
// There are cases when you want to block merging and for that you will need to provide
// params.HeadBranchExpectedSHA which will be compared with the latest sha from head branch
// if they are not the same error will be returned.
//
//nolint:gocognit,gocyclo,cyclop
func (s *Service) Merge(ctx context.Context, params *MergeParams) (MergeOutput, error) {
	err := params.Validate()
	if err != nil {
		return MergeOutput{}, fmt.Errorf("params not valid: %w", err)
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	// prepare the merge method function

	mergeMethod, ok := params.Method.Sanitize()
	if !ok && params.Method != "" {
		return MergeOutput{}, errors.InvalidArgument("Unsupported merge method: %q", params.Method)
	}

	var mergeFunc merge.Func

	switch mergeMethod {
	case enum.MergeMethodMerge:
		mergeFunc = merge.Merge
	case enum.MergeMethodSquash:
		mergeFunc = merge.Squash
	case enum.MergeMethodRebase:
		mergeFunc = merge.Rebase
	case enum.MergeMethodFastForward:
		mergeFunc = merge.FastForward
	default:
		// should not happen, the call to Sanitize above should handle this case.
		panic(fmt.Sprintf("unsupported merge method: %q", mergeMethod))
	}

	// find the commit SHAs

	baseCommitSHA := params.BaseSHA
	if baseCommitSHA.IsEmpty() {
		baseCommitSHA, err = s.git.ResolveRev(ctx, repoPath, api.EnsureBranchPrefix(params.BaseBranch))
		if err != nil {
			return MergeOutput{}, fmt.Errorf("failed to get base branch commit SHA: %w", err)
		}
	}

	headCommitSHA := params.HeadSHA
	if headCommitSHA.IsEmpty() {
		headCommitSHA, err = s.git.ResolveRev(ctx, repoPath, api.EnsureBranchPrefix(params.HeadBranch))
		if err != nil {
			return MergeOutput{}, fmt.Errorf("failed to get head branch commit SHA: %w", err)
		}

		if !params.HeadBranchExpectedSHA.IsEmpty() && !params.HeadBranchExpectedSHA.Equal(headCommitSHA) {
			return MergeOutput{}, errors.PreconditionFailed(
				"head branch '%s' is on SHA '%s' which doesn't match expected SHA '%s'.",
				params.HeadBranch,
				headCommitSHA,
				params.HeadBranchExpectedSHA)
		}
	}

	mergeBaseCommitSHA, _, err := s.git.GetMergeBase(ctx, repoPath, "origin",
		baseCommitSHA.String(), headCommitSHA.String())
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to get merge base: %w", err)
	}

	if headCommitSHA.Equal(mergeBaseCommitSHA) {
		return MergeOutput{}, errors.InvalidArgument("head branch doesn't contain any new commits.")
	}

	// find short stat and number of commits

	shortStat, err := s.git.DiffShortStat(
		ctx,
		repoPath,
		baseCommitSHA.String(),
		headCommitSHA.String(),
		true,
		false,
	)
	if err != nil {
		return MergeOutput{}, errors.Internal(err,
			"failed to find short stat between %s and %s", baseCommitSHA, headCommitSHA)
	}

	commitCount, err := merge.CommitCount(ctx, repoPath, baseCommitSHA.String(), headCommitSHA.String())
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to find commit count for merge check: %w", err)
	}

	// author and committer

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

	// create merge commit and update the references

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to create reference updater: %w", err)
	}

	var mergeCommitSHA sha.SHA
	var conflicts []string

	err = sharedrepo.Run(ctx, refUpdater, s.sharedRepoRoot, repoPath, func(s *sharedrepo.SharedRepo) error {
		message := parser.CleanUpWhitespace(params.Message)

		mergeCommitSHA, conflicts, err = mergeFunc(
			ctx,
			s,
			merge.Params{
				Author:       &author,
				Committer:    &committer,
				Message:      message,
				MergeBaseSHA: mergeBaseCommitSHA,
				TargetSHA:    baseCommitSHA,
				SourceSHA:    headCommitSHA,
			})
		if err != nil {
			return fmt.Errorf("failed to create merge commit: %w", err)
		}

		if mergeCommitSHA.IsEmpty() || len(conflicts) > 0 {
			return refUpdater.Init(ctx, nil) // update nothing
		}

		refUpdates := make([]hook.ReferenceUpdate, len(params.Refs))
		for i, ref := range params.Refs {
			oldValue := ref.Old
			newValue := ref.New

			if newValue.IsEmpty() { // replace all empty new values to the result of the merge
				newValue = mergeCommitSHA
			}

			refUpdates[i] = hook.ReferenceUpdate{
				Ref: ref.Name,
				Old: oldValue,
				New: newValue,
			}
		}

		err = refUpdater.Init(ctx, refUpdates)
		if err != nil {
			return fmt.Errorf("failed to init values of references (%v): %w", refUpdates, err)
		}

		return nil
	})
	if errors.IsConflict(err) {
		return MergeOutput{}, fmt.Errorf("failed to merge %q to %q in %q using the %q merge method: %w",
			params.HeadBranch, params.BaseBranch, params.RepoUID, mergeMethod, err)
	}
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to merge %q to %q in %q using the %q merge method: %w",
			params.HeadBranch, params.BaseBranch, params.RepoUID, mergeMethod, err)
	}
	if len(conflicts) > 0 {
		return MergeOutput{
			BaseSHA:          baseCommitSHA,
			HeadSHA:          headCommitSHA,
			MergeBaseSHA:     mergeBaseCommitSHA,
			MergeSHA:         sha.None,
			CommitCount:      commitCount,
			ChangedFileCount: shortStat.Files,
			Additions:        shortStat.Additions,
			Deletions:        shortStat.Deletions,
			ConflictFiles:    conflicts,
		}, nil
	}

	return MergeOutput{
		BaseSHA:          baseCommitSHA,
		HeadSHA:          headCommitSHA,
		MergeBaseSHA:     mergeBaseCommitSHA,
		MergeSHA:         mergeCommitSHA,
		CommitCount:      commitCount,
		ChangedFileCount: shortStat.Files,
		Additions:        shortStat.Additions,
		Deletions:        shortStat.Deletions,
		ConflictFiles:    nil,
	}, nil
}

type MergeBaseParams struct {
	ReadParams
	Ref1 string
	Ref2 string
}

func (p *MergeBaseParams) Validate() error {
	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if p.Ref1 == "" {
		// needs better naming
		return errors.InvalidArgument("first reference cannot be empty")
	}

	if p.Ref2 == "" {
		// needs better naming
		return errors.InvalidArgument("second reference cannot be empty")
	}

	return nil
}

type MergeBaseOutput struct {
	MergeBaseSHA sha.SHA
}

func (s *Service) MergeBase(
	ctx context.Context,
	params MergeBaseParams,
) (MergeBaseOutput, error) {
	if err := params.Validate(); err != nil {
		return MergeBaseOutput{}, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	result, _, err := s.git.GetMergeBase(ctx, repoPath, "", params.Ref1, params.Ref2)
	if err != nil {
		return MergeBaseOutput{}, err
	}

	return MergeBaseOutput{
		MergeBaseSHA: result,
	}, nil
}

type IsAncestorParams struct {
	ReadParams
	AncestorCommitSHA   sha.SHA
	DescendantCommitSHA sha.SHA
}

type IsAncestorOutput struct {
	Ancestor bool
}

func (s *Service) IsAncestor(
	ctx context.Context,
	params IsAncestorParams,
) (IsAncestorOutput, error) {
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	result, err := s.git.IsAncestor(
		ctx,
		repoPath,
		params.AlternateObjectDirs,
		params.AncestorCommitSHA,
		params.DescendantCommitSHA,
	)
	if err != nil {
		return IsAncestorOutput{}, err
	}

	return IsAncestorOutput{
		Ancestor: result,
	}, nil
}
