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
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/merge"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

// MergeParams is input structure object for merging operation.
type MergeParams struct {
	WriteParams
	BaseBranch string
	// HeadRepoUID specifies the UID of the repo that contains the head branch (required for forking).
	// WARNING: This field is currently not supported yet!
	HeadRepoUID string
	HeadBranch  string
	Title       string
	Message     string

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

	RefType enum.RefType
	RefName string

	// HeadExpectedSHA is commit sha on the head branch, if HeadExpectedSHA is older
	// than the HeadBranch latest sha then merge will fail.
	HeadExpectedSHA sha.SHA

	Force            bool
	DeleteHeadBranch bool

	Method enum.MergeMethod
}

func (p *MergeParams) Validate() error {
	if err := p.WriteParams.Validate(); err != nil {
		return err
	}

	if p.BaseBranch == "" {
		return errors.InvalidArgument("base branch is mandatory")
	}

	if p.HeadBranch == "" {
		return errors.InvalidArgument("head branch is mandatory")
	}

	if p.RefType != enum.RefTypeUndefined && p.RefName == "" {
		return errors.InvalidArgument("ref name has to be provided if type is defined")
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
// params.HeadExpectedSHA which will be compared with the latest sha from head branch
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
		return MergeOutput{}, errors.InvalidArgument("Unsupported merge method: %s", params.Method)
	}

	var mergeFunc merge.Func

	switch mergeMethod {
	case enum.MergeMethodMerge:
		mergeFunc = merge.Merge
	case enum.MergeMethodSquash:
		mergeFunc = merge.Squash
	case enum.MergeMethodRebase:
		mergeFunc = merge.Rebase
	default:
		// should not happen, the call to Sanitize above should handle this case.
		panic("unsupported merge method")
	}

	// set up the target reference

	var refPath string
	var refOldValue sha.SHA

	if params.RefType != enum.RefTypeUndefined {
		refPath, err = GetRefPath(params.RefName, params.RefType)
		if err != nil {
			return MergeOutput{}, fmt.Errorf(
				"failed to generate full reference for type '%s' and name '%s' for merge operation: %w",
				params.RefType, params.RefName, err)
		}

		refOldValue, err = s.git.GetFullCommitID(ctx, repoPath, refPath)
		if errors.IsNotFound(err) {
			refOldValue = sha.Nil
		} else if err != nil {
			return MergeOutput{}, fmt.Errorf("failed to resolve %q: %w", refPath, err)
		}
	}

	// logger

	log := log.Ctx(ctx).With().
		Str("repo_uid", params.RepoUID).
		Str("head", params.HeadBranch).
		Str("base", params.BaseBranch).
		Str("method", string(mergeMethod)).
		Str("ref", refPath).
		Logger()

	// find the commit SHAs

	baseCommitSHA, err := s.git.GetFullCommitID(ctx, repoPath, params.BaseBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to get merge base branch commit SHA: %w", err)
	}

	headCommitSHA, err := s.git.GetFullCommitID(ctx, repoPath, params.HeadBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to get merge base branch commit SHA: %w", err)
	}

	if !params.HeadExpectedSHA.IsEmpty() && !params.HeadExpectedSHA.Equal(headCommitSHA) {
		return MergeOutput{}, errors.PreconditionFailed(
			"head branch '%s' is on SHA '%s' which doesn't match expected SHA '%s'.",
			params.HeadBranch,
			headCommitSHA,
			params.HeadExpectedSHA)
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

	shortStat, err := s.git.DiffShortStat(ctx, repoPath, baseCommitSHA.String(), headCommitSHA.String(), true)
	if err != nil {
		return MergeOutput{}, errors.Internal(err,
			"failed to find short stat between %s and %s", baseCommitSHA, headCommitSHA)
	}

	commitCount, err := merge.CommitCount(ctx, repoPath, baseCommitSHA.String(), headCommitSHA.String())
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to find commit count for merge check: %w", err)
	}

	// handle simple merge check

	if params.RefType == enum.RefTypeUndefined {
		_, _, conflicts, err := merge.FindConflicts(ctx, repoPath, baseCommitSHA.String(), headCommitSHA.String())
		if err != nil {
			return MergeOutput{}, errors.Internal(err,
				"Merge check failed to find conflicts between commits %s and %s",
				baseCommitSHA.String(), headCommitSHA.String())
		}

		log.Debug().Msg("merged check completed")

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

	// merge message

	mergeMsg := strings.TrimSpace(params.Title)
	if len(params.Message) > 0 {
		mergeMsg += "\n\n" + strings.TrimSpace(params.Message)
	}

	// merge

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath, refPath)
	if err != nil {
		return MergeOutput{}, errors.Internal(err, "failed to create ref updater object")
	}

	if err := refUpdater.InitOld(ctx, refOldValue); err != nil {
		return MergeOutput{}, errors.Internal(err, "failed to set old reference value for ref updater")
	}

	mergeCommitSHA, conflicts, err := mergeFunc(
		ctx,
		refUpdater,
		repoPath, s.tmpDir,
		&author, &committer,
		mergeMsg,
		mergeBaseCommitSHA, baseCommitSHA, headCommitSHA)
	if err != nil {
		return MergeOutput{}, errors.Internal(err, "failed to merge %q to %q in %q using the %q merge method",
			params.HeadBranch, params.BaseBranch, params.RepoUID, mergeMethod)
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
