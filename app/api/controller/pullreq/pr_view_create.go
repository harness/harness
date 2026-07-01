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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	gittypes "github.com/harness/gitness/git/api"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

type PullReqViewCreateInput struct {
	SourceSHA    sha.SHA                       `json:"source_sha"`
	MergeBaseSHA sha.SHA                       `json:"merge_base_sha"`
	Groups       []PullReqViewCreateInputGroup `json:"groups"`
}

type PullReqViewCreateInputGroup struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Files       []string          `json:"files"`
}

func (f *PullReqViewCreateInput) Validate() error {
	seenTitles := make(map[string]struct{}, len(f.Groups))

	for i := range f.Groups {
		group := &f.Groups[i]

		if strings.TrimSpace(group.Title) == "" {
			return usererror.BadRequest("group title can't be empty")
		}

		if len(group.Files) == 0 {
			return usererror.BadRequestf("group %q must contain at least one file", group.Title)
		}

		if _, ok := seenTitles[group.Title]; ok {
			return usererror.BadRequestf("duplicate group title %q in request", group.Title)
		}
		seenTitles[group.Title] = struct{}{}

		// Validate tags (optional validation for reasonable constraints)
		for key := range group.Tags {
			if strings.TrimSpace(key) == "" {
				return usererror.BadRequestf("group %q has empty tag key", group.Title)
			}
		}
	}

	return nil
}

// PullReqViewCreate replaces all pull request file groups and their file assignments in one request.
func (c *Controller) PullReqViewCreate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *PullReqViewCreateInput,
) error {
	if err := in.Validate(); err != nil {
		return err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	// Resolve SHAs with precedence:
	// request payload -> PR fields -> git lookup for any remaining missing value.

	currentSourceSHA := in.SourceSHA.String()
	currentMergeBaseSHA := in.MergeBaseSHA.String()

	currentSourceSHA, currentMergeBaseSHA, err = c.resolveCurrentPRSHAs(
		ctx,
		repo,
		pr,
		currentSourceSHA,
		currentMergeBaseSHA,
	)
	if err != nil {
		return err
	}

	// Validate that the provided SHAs are part of the pull request diff and
	// collect the SHAs for requested file paths.

	requestedPaths := collectRequestedGroupPaths(in.Groups)
	pathToSHAs, err := c.resolveFileGroupPathSHAsFromPRDiff(
		ctx,
		repo,
		currentSourceSHA,
		currentMergeBaseSHA,
		requestedPaths,
	)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	err = c.tx.WithTx(ctx, func(txCtx context.Context) error {
		groups := make([]*types.PullReqFileGroupWithFiles, 0, len(in.Groups))

		for _, groupIn := range in.Groups {
			groupFiles, groupFilesErr := buildFileGroupFiles(groupIn.Files, pathToSHAs)
			if groupFilesErr != nil {
				return groupFilesErr
			}

			groups = append(groups, &types.PullReqFileGroupWithFiles{
				PullReqFileGroup: types.PullReqFileGroup{
					PullReqID:   pr.ID,
					Title:       groupIn.Title,
					Description: groupIn.Description,
					Tags:        groupIn.Tags,
					Created:     now,
					Updated:     now,
					CreatedBy:   session.Principal.ID,
					UpdatedBy:   session.Principal.ID,
				},
				Files: groupFiles,
			})
		}

		if deleteManyErr := c.fileGroupStore.DeleteByPrID(txCtx, pr.ID); deleteManyErr != nil {
			return fmt.Errorf("failed to delete pull request file groups in db: %w", deleteManyErr)
		}

		if createManyErr := c.fileGroupStore.CreateMany(txCtx, groups); createManyErr != nil {
			return fmt.Errorf("failed to create pull request file groups in db: %w", createManyErr)
		}

		return nil
	}, dbtx.TxDefault)
	if err != nil {
		return err
	}

	return nil
}

func collectRequestedGroupPaths(groups []PullReqViewCreateInputGroup) []string {
	paths := make([]string, 0)

	for _, group := range groups {
		paths = append(paths, group.Files...)
	}

	slices.Sort(paths)
	paths = slices.Compact(paths)

	return paths
}

type fileGroupPathSHAs struct {
	oldSHA string
	newSHA string
}

func (c *Controller) resolveFileGroupPathSHAsFromPRDiff(
	ctx context.Context,
	repo git.Repository,
	sourceSHA string,
	mergeBaseSHA string,
	paths []string,
) (map[string]fileGroupPathSHAs, error) {
	if len(paths) == 0 {
		return map[string]fileGroupPathSHAs{}, nil
	}

	diffRequests := make([]gittypes.FileDiffRequest, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			return nil, usererror.BadRequest("files.path is required")
		}
		diffRequests = append(diffRequests, gittypes.FileDiffRequest{Path: path})
	}

	stream := git.NewStreamReader(c.git.Diff(ctx, &git.DiffParams{
		ReadParams: git.CreateReadParams(repo),
		BaseRef:    mergeBaseSHA,
		HeadRef:    sourceSHA,
		MergeBase:  true,
	}, diffRequests...))

	pathToSHAs := make(map[string]fileGroupPathSHAs, len(paths))
	for {
		fileDiff, nextErr := stream.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("failed to stream pull request diff: %w", nextErr)
		}

		pathToSHAs[fileDiff.Path] = fileGroupPathSHAs{oldSHA: fileDiff.OldSHA, newSHA: fileDiff.SHA}
	}

	return pathToSHAs, nil
}

func (c *Controller) resolveCurrentPRSHAsFromGit(
	ctx context.Context,
	repo *types.RepositoryCore,
	pr *types.PullReq,
	sourceSHA string,
	mergeBaseSHA string,
) (string, string, error) {
	readParams := git.CreateReadParams(repo)

	if sourceSHA == "" {
		headRef, err := c.git.GetRef(ctx, git.GetRefParams{
			ReadParams: readParams,
			Name:       strconv.FormatInt(pr.Number, 10),
			Type:       gitenum.RefTypePullReqHead,
		})
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve pull request head reference: %w", err)
		}

		sourceSHA = headRef.SHA.String()
	}

	if mergeBaseSHA == "" {
		targetRef, err := c.git.GetRef(ctx, git.GetRefParams{
			ReadParams: readParams,
			Name:       pr.TargetBranch,
			Type:       gitenum.RefTypeBranch,
		})
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve target branch reference: %w", err)
		}

		mergeBaseInfo, err := c.git.MergeBase(ctx, git.MergeBaseParams{
			ReadParams: readParams,
			Ref1:       sourceSHA,
			Ref2:       targetRef.SHA.String(),
		})
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve merge base for pull request: %w", err)
		}

		mergeBaseSHA = mergeBaseInfo.MergeBaseSHA.String()
	}

	return sourceSHA, mergeBaseSHA, nil
}

func (c *Controller) resolveCurrentPRSHAs(
	ctx context.Context,
	repo *types.RepositoryCore,
	pr *types.PullReq,
	sourceSHA string,
	mergeBaseSHA string,
) (string, string, error) {
	if sourceSHA != "" && mergeBaseSHA != "" {
		return sourceSHA, mergeBaseSHA, nil
	}

	if sourceSHA == "" {
		sourceSHA = pr.SourceSHA
	}
	if mergeBaseSHA == "" {
		mergeBaseSHA = pr.MergeBaseSHA
	}

	if sourceSHA != "" && mergeBaseSHA != "" {
		return sourceSHA, mergeBaseSHA, nil
	}

	resolvedSourceSHA, resolvedMergeBaseSHA, resolveErr := c.resolveCurrentPRSHAsFromGit(
		ctx,
		repo,
		pr,
		sourceSHA,
		mergeBaseSHA,
	)
	if resolveErr != nil {
		return "", "", fmt.Errorf("failed to resolve pull request SHAs from git: %w", resolveErr)
	}

	if sourceSHA == "" {
		sourceSHA = resolvedSourceSHA
	}
	if mergeBaseSHA == "" {
		mergeBaseSHA = resolvedMergeBaseSHA
	}

	return sourceSHA, mergeBaseSHA, nil
}

func buildFileGroupFiles(
	paths []string,
	pathToSHAs map[string]fileGroupPathSHAs,
) ([]*types.PullReqFileGroupFile, error) {
	files := make([]*types.PullReqFileGroupFile, 0, len(paths))
	emittedPaths := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		if path == "" {
			return nil, usererror.BadRequest("files.path is required")
		}

		if _, ok := emittedPaths[path]; ok {
			continue
		}

		pathSHAs, ok := pathToSHAs[path]
		if !ok {
			return nil, usererror.BadRequestf(
				"path %q is not part of the pull request diff", path,
			)
		}

		emittedPaths[path] = struct{}{}
		files = append(files, &types.PullReqFileGroupFile{
			Path:   path,
			OldSHA: pathSHAs.oldSHA,
			NewSHA: pathSHAs.newSHA,
		})
	}

	return files, nil
}
