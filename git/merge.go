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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/tempdir"
	"github.com/harness/gitness/git/types"

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
	HeadExpectedSHA string

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
	BaseSHA string
	// HeadSHA is the sha of the latest commit on the head branch that was used for merging.
	HeadSHA string
	// MergeBaseSHA is the sha of the merge base of the HeadSHA and BaseSHA
	MergeBaseSHA string
	// MergeSHA is the sha of the commit after merging HeadSHA with BaseSHA.
	MergeSHA string

	CommitCount      int
	ChangedFileCount int
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
	if err := params.Validate(); err != nil {
		return MergeOutput{}, fmt.Errorf("Merge: params not valid: %w", err)
	}

	log := log.Ctx(ctx).With().Str("repo_uid", params.RepoUID).Logger()

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	baseBranch := "base"
	trackingBranch := "tracking"

	pr := &types.PullRequest{
		BaseRepoPath: repoPath,
		BaseBranch:   params.BaseBranch,
		HeadBranch:   params.HeadBranch,
	}

	log.Debug().Msg("create temporary repository")

	// Clone base repo.
	tmpRepo, err := s.adapter.CreateTemporaryRepoForPR(ctx, s.tmpDir, pr, baseBranch, trackingBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("Merge: failed to initialize temporary repo: %w", err)
	}
	defer func() {
		rmErr := tempdir.RemoveTemporaryPath(tmpRepo.Path)
		if rmErr != nil {
			log.Warn().Msgf("Removing temporary location %s for merge operation was not successful", tmpRepo.Path)
		}
	}()

	log.Debug().Msg("get merge base")

	mergeBaseCommitSHA, _, err := s.adapter.GetMergeBase(ctx, tmpRepo.Path, "origin", baseBranch, trackingBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to get merge base: %w", err)
	}

	if tmpRepo.HeadSHA == mergeBaseCommitSHA {
		return MergeOutput{}, errors.InvalidArgument("no changes between head branch %s and base branch %s",
			params.HeadBranch, params.BaseBranch)
	}

	if params.HeadExpectedSHA != "" && params.HeadExpectedSHA != tmpRepo.HeadSHA {
		return MergeOutput{}, errors.PreconditionFailed(
			"head branch '%s' is on SHA '%s' which doesn't match expected SHA '%s'.",
			params.HeadBranch,
			tmpRepo.HeadSHA,
			params.HeadExpectedSHA)
	}

	log.Debug().Msg("get diff tree")

	var outbuf, errbuf strings.Builder
	// Enable sparse-checkout
	sparseCheckoutList, err := s.adapter.GetDiffTree(ctx, tmpRepo.Path, baseBranch, trackingBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("execution of GetDiffTree failed: %w", err)
	}

	log.Debug().Msg("prepare sparse-checkout")

	infoPath := filepath.Join(tmpRepo.Path, ".git", "info")
	if err = os.MkdirAll(infoPath, 0o700); err != nil {
		return MergeOutput{}, fmt.Errorf("unable to create .git/info in tmpRepo.Path: %w", err)
	}

	sparseCheckoutListPath := filepath.Join(infoPath, "sparse-checkout")
	if err = os.WriteFile(sparseCheckoutListPath, []byte(sparseCheckoutList), 0o600); err != nil {
		return MergeOutput{},
			fmt.Errorf("unable to write .git/info/sparse-checkout file in tmpRepo.Path: %w", err)
	}

	log.Debug().Msg("get diff stats")

	shortStat, err := s.adapter.DiffShortStat(ctx, tmpRepo.Path, tmpRepo.BaseSHA, tmpRepo.HeadSHA, true)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("execution of DiffShortStat failed: %w", err)
	}
	changedFileCount := shortStat.Files

	log.Debug().Msg("get commit divergene")

	divergences, err := s.adapter.GetCommitDivergences(ctx, tmpRepo.Path,
		[]types.CommitDivergenceRequest{{From: tmpRepo.HeadSHA, To: tmpRepo.BaseSHA}}, 0)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("execution of GetCommitDivergences failed: %w", err)
	}
	commitCount := int(divergences[0].Ahead)

	log.Debug().Msg("update git configuration")

	// Switch off LFS process (set required, clean and smudge here also)
	if err = s.adapter.Config(ctx, tmpRepo.Path, "filter.lfs.process", ""); err != nil {
		return MergeOutput{}, err
	}

	if err = s.adapter.Config(ctx, tmpRepo.Path, "filter.lfs.required", "false"); err != nil {
		return MergeOutput{}, err
	}

	if err = s.adapter.Config(ctx, tmpRepo.Path, "filter.lfs.clean", ""); err != nil {
		return MergeOutput{}, err
	}

	if err = s.adapter.Config(ctx, tmpRepo.Path, "filter.lfs.smudge", ""); err != nil {
		return MergeOutput{}, err
	}

	if err = s.adapter.Config(ctx, tmpRepo.Path, "core.sparseCheckout", "true"); err != nil {
		return MergeOutput{}, err
	}

	log.Debug().Msg("read tree")

	// Read base branch index
	if err = s.adapter.ReadTree(ctx, tmpRepo.Path, "HEAD", io.Discard); err != nil {
		return MergeOutput{}, fmt.Errorf("failed to read tree: %w", err)
	}
	outbuf.Reset()
	errbuf.Reset()

	committer := params.Actor
	if params.Committer != nil {
		committer = *params.Committer
	}
	committerDate := time.Now().UTC()
	if params.CommitterDate != nil {
		committerDate = *params.CommitterDate
	}

	author := committer
	if params.Author != nil {
		author = *params.Author
	}
	authorDate := committerDate
	if params.AuthorDate != nil {
		authorDate = *params.AuthorDate
	}

	// Because this may call hooks we should pass in the environment
	// TODO: merge specific envars should be set by the adapter impl.
	env := append(CreateEnvironmentForPush(ctx, params.WriteParams),
		"GIT_AUTHOR_NAME="+author.Name,
		"GIT_AUTHOR_EMAIL="+author.Email,
		"GIT_AUTHOR_DATE="+authorDate.Format(time.RFC3339),
		"GIT_COMMITTER_NAME="+committer.Name,
		"GIT_COMMITTER_EMAIL="+committer.Email,
		"GIT_COMMITTER_DATE="+committerDate.Format(time.RFC3339),
	)

	mergeMsg := strings.TrimSpace(params.Title)
	if len(params.Message) > 0 {
		mergeMsg += "\n\n" + strings.TrimSpace(params.Message)
	}

	if params.Method == "" {
		params.Method = enum.MergeMethodMerge
	}

	log.Debug().Msg("perform merge")

	result, err := s.adapter.Merge(
		ctx,
		pr,
		params.Method,
		baseBranch,
		trackingBranch,
		tmpRepo.Path,
		mergeMsg,
		&types.Identity{
			Name:  author.Name,
			Email: author.Email,
		},
		env...)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("merge failed: %w", err)
	}

	if len(result.ConflictFiles) > 0 {
		return MergeOutput{
			BaseSHA:          tmpRepo.BaseSHA,
			HeadSHA:          tmpRepo.HeadSHA,
			MergeBaseSHA:     mergeBaseCommitSHA,
			MergeSHA:         "",
			CommitCount:      commitCount,
			ChangedFileCount: changedFileCount,
			ConflictFiles:    result.ConflictFiles,
		}, nil
	}

	log.Debug().Msg("get commit id")

	mergeCommitSHA, err := s.adapter.GetFullCommitID(ctx, tmpRepo.Path, baseBranch)
	if err != nil {
		return MergeOutput{}, fmt.Errorf("failed to get full commit id for the new merge: %w", err)
	}

	if params.RefType == enum.RefTypeUndefined {
		log.Debug().Msg("done (merge-check only)")

		return MergeOutput{
			BaseSHA:          tmpRepo.BaseSHA,
			HeadSHA:          tmpRepo.HeadSHA,
			MergeBaseSHA:     mergeBaseCommitSHA,
			MergeSHA:         mergeCommitSHA,
			CommitCount:      commitCount,
			ChangedFileCount: changedFileCount,
			ConflictFiles:    nil,
		}, nil
	}

	refPath, err := GetRefPath(params.RefName, params.RefType)
	if err != nil {
		return MergeOutput{}, fmt.Errorf(
			"failed to generate full reference for type '%s' and name '%s' for merge operation: %w",
			params.RefType, params.RefName, err)
	}
	pushRef := baseBranch + ":" + refPath

	log.Debug().Msg("push to original repo")

	if err = s.adapter.Push(ctx, tmpRepo.Path, types.PushOptions{
		Remote: "origin",
		Branch: pushRef,
		Force:  params.Force,
		Env:    env,
	}); err != nil {
		return MergeOutput{}, fmt.Errorf("failed to push merge commit to ref '%s': %w", refPath, err)
	}

	log.Debug().Msg("done")

	return MergeOutput{
		BaseSHA:          tmpRepo.BaseSHA,
		HeadSHA:          tmpRepo.HeadSHA,
		MergeBaseSHA:     mergeBaseCommitSHA,
		MergeSHA:         mergeCommitSHA,
		CommitCount:      commitCount,
		ChangedFileCount: changedFileCount,
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
	MergeBaseSHA string
}

func (s *Service) MergeBase(
	ctx context.Context,
	params MergeBaseParams,
) (MergeBaseOutput, error) {
	if err := params.Validate(); err != nil {
		return MergeBaseOutput{}, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	result, _, err := s.adapter.GetMergeBase(ctx, repoPath, "", params.Ref1, params.Ref2)
	if err != nil {
		return MergeBaseOutput{}, err
	}

	return MergeBaseOutput{
		MergeBaseSHA: result,
	}, nil
}

type IsAncestorParams struct {
	ReadParams
	AncestorCommitSHA   string
	DescendantCommitSHA string
}

type IsAncestorOutput struct {
	Ancestor bool
}

func (s *Service) IsAncestor(
	ctx context.Context,
	params IsAncestorParams,
) (IsAncestorOutput, error) {
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	result, err := s.adapter.IsAncestor(ctx, repoPath, params.AncestorCommitSHA, params.DescendantCommitSHA)
	if err != nil {
		return IsAncestorOutput{}, err
	}

	return IsAncestorOutput{
		Ancestor: result,
	}, nil
}
