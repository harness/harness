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

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

// GetLatestCommit gets the latest commit of a path relative from the provided revision.
func (a Adapter) GetLatestCommit(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) (*types.Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	return getCommit(ctx, repoPath, rev, treePath)
}

func getGiteaCommits(
	giteaRepo *gitea.Repository,
	commitIDs []string,
) ([]*gitea.Commit, error) {
	var giteaCommits []*gitea.Commit
	if len(commitIDs) == 0 {
		return giteaCommits, nil
	}

	for _, commitID := range commitIDs {
		commit, err := giteaRepo.GetCommit(commitID)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit '%s': %w", commitID, err)
		}
		giteaCommits = append(giteaCommits, commit)
	}

	return giteaCommits, nil
}

func (a Adapter) listCommitSHAs(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	filter types.CommitFilter,
) ([]string, error) {
	args := make([]string, 0, 16)
	args = append(args, "rev-list")

	// return commits only up to a certain reference if requested
	if filter.AfterRef != "" {
		// ^REF tells the rev-list command to return only commits that aren't reachable by SHA
		args = append(args, fmt.Sprintf("^%s", filter.AfterRef))
	}
	// add refCommitSHA as starting point
	args = append(args, ref)

	if len(filter.Path) != 0 {
		args = append(args, "--", filter.Path)
	}

	// add pagination if requested
	// TODO: we should add absolut limits to protect git (return error)
	if limit > 0 {
		args = append(args, "--max-count", fmt.Sprint(limit))

		if page > 1 {
			args = append(args, "--skip", fmt.Sprint((page-1)*limit))
		}
	}

	if filter.Since > 0 || filter.Until > 0 {
		args = append(args, "--date", "unix")
	}
	if filter.Since > 0 {
		args = append(args, "--since", strconv.FormatInt(filter.Since, 10))
	}
	if filter.Until > 0 {
		args = append(args, "--until", strconv.FormatInt(filter.Until, 10))
	}
	if filter.Committer != "" {
		args = append(args, "--committer", filter.Committer)
	}

	stdout, _, runErr := gitea.NewCommand(ctx, args...).RunStdBytes(&gitea.RunOpts{Dir: repoPath})
	if runErr != nil {
		// TODO: handle error in case they don't have a common merge base!
		return nil, processGiteaErrorf(runErr, "failed to trigger rev-list command")
	}

	return parseLinesToSlice(stdout), nil
}

// ListCommitSHAs lists the commits reachable from ref.
// Note: ref & afterRef can be Branch / Tag / CommitSHA.
// Note: commits returned are [ref->...->afterRef).
func (a Adapter) ListCommitSHAs(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	filter types.CommitFilter,
) ([]string, error) {
	return a.listCommitSHAs(ctx, repoPath, ref, page, limit, filter)
}

// ListCommits lists the commits reachable from ref.
// Note: ref & afterRef can be Branch / Tag / CommitSHA.
// Note: commits returned are [ref->...->afterRef).
func (a Adapter) ListCommits(ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	filter types.CommitFilter,
) ([]types.Commit, []types.PathRenameDetails, error) {
	if repoPath == "" {
		return nil, nil, ErrRepositoryPathEmpty
	}
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	commitSHAs, err := a.listCommitSHAs(ctx, repoPath, ref, page, limit, filter)
	if err != nil {
		return nil, nil, err
	}

	giteaCommits, err := getGiteaCommits(giteaRepo, commitSHAs)
	if err != nil {
		return nil, nil, err
	}

	commits := make([]types.Commit, len(giteaCommits))
	for i := range giteaCommits {
		var commit *types.Commit
		commit, err = mapGiteaCommit(giteaCommits[i])
		if err != nil {
			return nil, nil, err
		}
		commits[i] = *commit
	}

	if len(filter.Path) != 0 {
		renameDetailsList, err := getRenameDetails(giteaRepo, commits, filter.Path)
		if err != nil {
			return nil, nil, err
		}
		cleanedUpCommits := cleanupCommitsForRename(commits, renameDetailsList, filter.Path)
		return cleanedUpCommits, renameDetailsList, nil
	}

	return commits, nil, nil
}

// In case of rename of a file, same commit will be listed twice - Once in old file and second time in new file.
// Hence, we are making it a pattern to only list it as part of new file and not as part of old file.
func cleanupCommitsForRename(
	commits []types.Commit,
	renameDetails []types.PathRenameDetails,
	path string,
) []types.Commit {
	if len(commits) == 0 {
		return commits
	}
	for _, renameDetail := range renameDetails {
		// Since rename details is present it implies that we have commits and hence don't need null check.
		if commits[0].SHA == renameDetail.CommitSHABefore && path == renameDetail.OldPath {
			return commits[1:]
		}
	}
	return commits
}

func getRenameDetails(
	giteaRepo *gitea.Repository,
	commits []types.Commit,
	path string,
) ([]types.PathRenameDetails, error) {
	if len(commits) == 0 {
		return []types.PathRenameDetails{}, nil
	}

	renameDetailsList := make([]types.PathRenameDetails, 0, 2)

	renameDetails, err := giteaGetRenameDetails(giteaRepo, commits[0].SHA, path)
	if err != nil {
		return nil, err
	}
	if renameDetails.NewPath != "" || renameDetails.OldPath != "" {
		renameDetails.CommitSHABefore = commits[0].SHA
		renameDetailsList = append(renameDetailsList, *renameDetails)
	}

	if len(commits) == 1 {
		return renameDetailsList, nil
	}

	renameDetailsLast, err := giteaGetRenameDetails(giteaRepo, commits[len(commits)-1].SHA, path)
	if err != nil {
		return nil, err
	}

	if renameDetailsLast.NewPath != "" || renameDetailsLast.OldPath != "" {
		renameDetailsLast.CommitSHAAfter = commits[len(commits)-1].SHA
		renameDetailsList = append(renameDetailsList, *renameDetailsLast)
	}
	return renameDetailsList, nil
}

func giteaGetRenameDetails(
	giteaRepo *gitea.Repository,
	ref string,
	path string,
) (*types.PathRenameDetails, error) {
	stdout, _, runErr := gitea.NewCommand(giteaRepo.Ctx, "log", ref, "--name-status", "--pretty=format:", "-1").
		RunStdBytes(&gitea.RunOpts{Dir: giteaRepo.Path})
	if runErr != nil {
		return nil, fmt.Errorf("failed to trigger log command: %w", runErr)
	}

	lines := parseLinesToSlice(stdout)

	changeType, oldPath, newPath, err := getFileChangeTypeFromLog(lines, path)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(*changeType, "R") {
		return &types.PathRenameDetails{
			OldPath: *oldPath,
			NewPath: *newPath,
		}, nil
	}

	return &types.PathRenameDetails{}, nil
}

func getFileChangeTypeFromLog(
	changeStrings []string,
	filePath string,
) (*string, *string, *string, error) {
	for _, changeString := range changeStrings {
		if strings.Contains(changeString, filePath) {
			changeInfo := strings.Split(changeString, "\t")
			if len(changeInfo) != 3 {
				return &changeInfo[0], nil, nil, nil
			}
			return &changeInfo[0], &changeInfo[1], &changeInfo[2], nil
		}
	}
	return nil, nil, nil, fmt.Errorf("could not parse change for the file '%s'", filePath)
}

// GetCommit returns the (latest) commit for a specific revision.
func (a Adapter) GetCommit(
	ctx context.Context,
	repoPath string,
	rev string,
) (*types.Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	return getCommit(ctx, repoPath, rev, "")
}

func (a Adapter) GetFullCommitID(
	ctx context.Context,
	repoPath string,
	shortID string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
	return gitea.GetFullCommitID(ctx, repoPath, shortID)
}

// GetCommits returns the (latest) commits for a specific list of refs.
// Note: ref can be Branch / Tag / CommitSHA.
func (a Adapter) GetCommits(
	ctx context.Context,
	repoPath string,
	refs []string,
) ([]types.Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	commits := make([]types.Commit, len(refs))
	for i, sha := range refs {
		var giteaCommit *gitea.Commit
		giteaCommit, err = giteaRepo.GetCommit(sha)
		if err != nil {
			return nil, processGiteaErrorf(err, "error getting commit '%s'", sha)
		}

		var commit *types.Commit
		commit, err = mapGiteaCommit(giteaCommit)
		if err != nil {
			return nil, err
		}
		commits[i] = *commit
	}

	return commits, nil
}

// GetCommitDivergences returns the count of the diverging commits for all branch pairs.
// IMPORTANT: If a max is provided it limits the overal count of diverging commits
// (max 10 could lead to (0, 10) while it's actually (2, 12)).
func (a Adapter) GetCommitDivergences(
	ctx context.Context,
	repoPath string,
	requests []types.CommitDivergenceRequest,
	max int32,
) ([]types.CommitDivergence, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	var err error
	res := make([]types.CommitDivergence, len(requests))
	for i, req := range requests {
		res[i], err = a.getCommitDivergence(ctx, repoPath, req, max)
		if types.IsNotFoundError(err) {
			res[i] = types.CommitDivergence{Ahead: -1, Behind: -1}
			continue
		}
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// getCommitDivergence returns the count of diverging commits for a pair of branches.
// IMPORTANT: If a max is provided it limits the overal count of diverging commits
// (max 10 could lead to (0, 10) while it's actually (2, 12)).
// NOTE: Gitea implementation makes two git cli calls, but it can be done with one
// (downside is the max behavior explained above).
func (a Adapter) getCommitDivergence(
	ctx context.Context,
	repoPath string,
	req types.CommitDivergenceRequest,
	max int32,
) (types.CommitDivergence, error) {
	// prepare args
	args := []string{
		"rev-list",
		"--count",
		"--left-right",
	}
	// limit count if requested.
	if max > 0 {
		args = append(args, "--max-count")
		args = append(args, fmt.Sprint(max))
	}
	// add query to get commits without shared base commits
	args = append(args, fmt.Sprintf("%s...%s", req.From, req.To))

	var err error
	cmd := gitea.NewCommand(ctx, args...)
	stdOut, stdErr, err := cmd.RunStdString(&gitea.RunOpts{Dir: repoPath})
	if err != nil {
		return types.CommitDivergence{},
			processGiteaErrorf(err, "git rev-list failed for '%s...%s' (stdErr: '%s')", req.From, req.To, stdErr)
	}

	// parse output, e.g.: `1       2\n`
	rawLeft, rawRight, ok := strings.Cut(stdOut, "\t")
	if !ok {
		return types.CommitDivergence{}, fmt.Errorf("git rev-list returned unexpected output '%s'", stdOut)
	}

	// trim any unnecessary characters
	rawLeft = strings.TrimRight(rawLeft, " \t")
	rawRight = strings.TrimRight(rawRight, " \t\n")

	// parse numbers
	left, err := strconv.ParseInt(rawLeft, 10, 32)
	if err != nil {
		return types.CommitDivergence{},
			fmt.Errorf("failed to parse git rev-list output for ahead '%s' (full: '%s')): %w", rawLeft, stdOut, err)
	}
	right, err := strconv.ParseInt(rawRight, 10, 32)
	if err != nil {
		return types.CommitDivergence{},
			fmt.Errorf("failed to parse git rev-list output for behind '%s' (full: '%s')): %w", rawRight, stdOut, err)
	}

	return types.CommitDivergence{
		Ahead:  int32(left),
		Behind: int32(right),
	}, nil
}

func parseLinesToSlice(output []byte) []string {
	if len(output) == 0 {
		return nil
	}

	lines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})

	slice := make([]string, len(lines))
	for i, line := range lines {
		slice[i] = string(line)
	}

	return slice
}

func getCommit(
	ctx context.Context,
	repoPath string,
	rev string,
	path string,
) (*types.Commit, error) {
	const format = "" +
		fmtCommitHash + fmtZero + // 0
		fmtAuthorName + fmtZero + // 1
		fmtAuthorEmail + fmtZero + // 2
		fmtAuthorTime + fmtZero + // 3
		fmtCommitterName + fmtZero + // 4
		fmtCommitterEmail + fmtZero + // 5
		fmtCommitterTime + fmtZero + // 6
		fmtSubject + fmtZero + // 7
		fmtBody // 8

	args := []string{"log", "--max-count=1", "--format=" + format, rev}
	if path != "" {
		args = append(args, "--", path)
	}

	commitLine, stderr, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath})
	if strings.Contains(stderr, "ambiguous argument") {
		return nil, errors.NotFound("revision %q not found", rev)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to run git to get commit data: %w", err)
	}

	if commitLine == "" {
		return nil, errors.InvalidArgument("path %q not found in %s", path, rev)
	}

	const columnCount = 9

	commitData := strings.Split(strings.TrimSpace(commitLine), separatorZero)
	if len(commitData) != columnCount {
		return nil, fmt.Errorf(
			"unexpected git log formatted output, expected %d, but got %d columns", columnCount, len(commitData))
	}

	sha := commitData[0]
	authorName := commitData[1]
	authorEmail := commitData[2]
	authorTimestamp := commitData[3]
	committerName := commitData[4]
	committerEmail := commitData[5]
	committerTimestamp := commitData[6]
	subject := commitData[7]
	body := commitData[8]

	authorTime, _ := time.Parse(time.RFC3339Nano, authorTimestamp)
	committerTime, _ := time.Parse(time.RFC3339Nano, committerTimestamp)

	return &types.Commit{
		SHA:     sha,
		Title:   subject,
		Message: body,
		Author: types.Signature{
			Identity: types.Identity{
				Name:  authorName,
				Email: authorEmail,
			},
			When: authorTime,
		},
		Committer: types.Signature{
			Identity: types.Identity{
				Name:  committerName,
				Email: committerEmail,
			},
			When: committerTime,
		},
	}, nil
}
