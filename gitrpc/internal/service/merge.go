// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/tempdir"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MergeService struct {
	rpc.UnimplementedMergeServiceServer
	adapter      GitAdapter
	reposRoot    string
	reposTempDir string
}

var _ rpc.MergeServiceServer = (*MergeService)(nil)

func NewMergeService(adapter GitAdapter, reposRoot, reposTempDir string) (*MergeService, error) {
	return &MergeService{
		adapter:      adapter,
		reposRoot:    reposRoot,
		reposTempDir: reposTempDir,
	}, nil
}

//nolint:funlen,gocognit // maybe some refactoring when we add fast forward merging
func (s MergeService) Merge(
	ctx context.Context,
	request *rpc.MergeRequest,
) (*rpc.MergeResponse, error) {
	if err := validateMergeRequest(request); err != nil {
		return nil, err
	}

	base := request.Base
	repoPath := getFullPathForRepo(s.reposRoot, base.RepoUid)

	pr := &types.PullRequest{
		BaseRepoPath: repoPath,
		BaseBranch:   request.BaseBranch,
		HeadBranch:   request.HeadBranch,
	}

	// Clone base repo.
	tmpBasePath, err := s.adapter.CreateTemporaryRepoForPR(ctx, s.reposTempDir, pr)
	if err != nil {
		return nil, err
	}
	defer func() {
		rmErr := tempdir.RemoveTemporaryPath(tmpBasePath)
		if rmErr != nil {
			log.Ctx(ctx).Warn().Msgf("Removing temporary location %s for merge operation was not successful", tmpBasePath)
		}
	}()

	// no error check needed, all branches were created when creating the temporary repo
	baseBranch := "base"
	trackingBranch := "tracking"
	headCommit, err := s.adapter.GetCommit(ctx, tmpBasePath, trackingBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit of tracking branch (head): %w", err)
	}
	headCommitSHA := headCommit.SHA
	baseCommit, err := s.adapter.GetCommit(ctx, tmpBasePath, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit of base branch: %w", err)
	}
	baseCommitSHA := baseCommit.SHA
	mergeBaseCommitSHA, _, err := s.adapter.GetMergeBase(ctx, tmpBasePath, "origin", baseBranch, trackingBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	if headCommitSHA == mergeBaseCommitSHA {
		return nil, ErrInvalidArgumentf("no changes between head branch %s and base branch %s",
			request.HeadBranch, request.BaseBranch)
	}

	if request.HeadExpectedSha != "" && request.HeadExpectedSha != headCommitSHA {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"head branch '%s' is on SHA '%s' which doesn't match expected SHA '%s'.",
			request.HeadBranch,
			headCommitSHA,
			request.HeadExpectedSha)
	}

	var outbuf, errbuf strings.Builder
	// Enable sparse-checkout
	sparseCheckoutList, err := s.adapter.GetDiffTree(ctx, tmpBasePath, baseBranch, trackingBranch)
	if err != nil {
		return nil, fmt.Errorf("execution of GetDiffTree failed: %w", err)
	}

	infoPath := filepath.Join(tmpBasePath, ".git", "info")
	if err = os.MkdirAll(infoPath, 0o700); err != nil {
		return nil, fmt.Errorf("unable to create .git/info in tmpBasePath: %w", err)
	}

	sparseCheckoutListPath := filepath.Join(infoPath, "sparse-checkout")
	if err = os.WriteFile(sparseCheckoutListPath, []byte(sparseCheckoutList), 0o600); err != nil {
		return nil,
			fmt.Errorf("unable to write .git/info/sparse-checkout file in tmpBasePath: %w", err)
	}

	// Switch off LFS process (set required, clean and smudge here also)
	if err = s.adapter.Config(ctx, tmpBasePath, "filter.lfs.process", ""); err != nil {
		return nil, err
	}

	if err = s.adapter.Config(ctx, tmpBasePath, "filter.lfs.required", "false"); err != nil {
		return nil, err
	}

	if err = s.adapter.Config(ctx, tmpBasePath, "filter.lfs.clean", ""); err != nil {
		return nil, err
	}

	if err = s.adapter.Config(ctx, tmpBasePath, "filter.lfs.smudge", ""); err != nil {
		return nil, err
	}

	if err = s.adapter.Config(ctx, tmpBasePath, "core.sparseCheckout", "true"); err != nil {
		return nil, err
	}

	// Read base branch index
	if err = s.adapter.ReadTree(ctx, tmpBasePath, "HEAD", io.Discard); err != nil {
		return nil, fmt.Errorf("failed to read tree: %w", err)
	}
	outbuf.Reset()
	errbuf.Reset()

	committer := base.Actor
	if request.Committer != nil {
		committer = request.Committer
	}
	author := committer
	if request.Author != nil {
		author = request.Author
	}

	timeStr := time.Now().Format(time.RFC3339)

	// Because this may call hooks we should pass in the environment
	env := append(CreateEnvironmentForPush(ctx, base),
		"GIT_AUTHOR_NAME="+author.Name,
		"GIT_AUTHOR_EMAIL="+author.Email,
		"GIT_AUTHOR_DATE="+timeStr,
		"GIT_COMMITTER_NAME="+committer.Name,
		"GIT_COMMITTER_EMAIL="+committer.Email,
		"GIT_COMMITTER_DATE="+timeStr,
	)

	mergeMsg := strings.TrimSpace(request.Title)
	if len(request.Message) > 0 {
		mergeMsg += "\n\n" + strings.TrimSpace(request.Message)
	}

	if err = s.adapter.Merge(
		ctx,
		pr,
		enum.MergeMethodFromRPC(request.Method),
		baseBranch,
		trackingBranch,
		tmpBasePath,
		mergeMsg,
		env,
		&types.Identity{
			Name:  author.Name,
			Email: author.Email,
		}); err != nil {
		return nil, processGitErrorf(err, "merge failed")
	}

	mergeCommitSHA, err := s.adapter.GetFullCommitID(ctx, tmpBasePath, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get full commit id for the new merge: %w", err)
	}

	refType := enum.RefFromRPC(request.RefType)
	if refType == enum.RefTypeUndefined {
		return &rpc.MergeResponse{
			BaseSha:      baseCommitSHA,
			HeadSha:      headCommitSHA,
			MergeBaseSha: mergeBaseCommitSHA,
			MergeSha:     mergeCommitSHA,
		}, nil
	}

	refPath, err := s.adapter.GetRefPath(request.RefName, refType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate full reference for type '%s' and name '%s' for merge operation: %w",
			request.RefType, request.RefName, err)
	}
	pushRef := baseBranch + ":" + refPath

	if err = s.adapter.Push(ctx, tmpBasePath, types.PushOptions{
		Remote: "origin",
		Branch: pushRef,
		Force:  request.Force,
		Env:    env,
	}); err != nil {
		return nil, fmt.Errorf("failed to push merge commit to ref '%s': %w", refPath, err)
	}

	return &rpc.MergeResponse{
		BaseSha:      baseCommitSHA,
		HeadSha:      headCommitSHA,
		MergeBaseSha: mergeBaseCommitSHA,
		MergeSha:     mergeCommitSHA,
	}, nil
}

func validateMergeRequest(request *rpc.MergeRequest) error {
	base := request.Base
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	author := base.Actor
	if author == nil {
		return fmt.Errorf("empty actor")
	}

	if len(author.Email) == 0 {
		return fmt.Errorf("empty user email")
	}

	if len(author.Name) == 0 {
		return fmt.Errorf("empty user name")
	}

	if len(request.BaseBranch) == 0 {
		return fmt.Errorf("empty branch name")
	}

	if len(request.HeadBranch) == 0 {
		return fmt.Errorf("empty head branch name")
	}

	if request.RefType != rpc.RefType_Undefined && len(request.RefName) == 0 {
		return fmt.Errorf("ref name has to be provided if type is defined")
	}

	return nil
}
