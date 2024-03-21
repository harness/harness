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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
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

	return GetCommit(ctx, repoPath, rev, treePath)
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
	cmd := command.New("rev-list")

	// return commits only up to a certain reference if requested
	if filter.AfterRef != "" {
		// ^REF tells the rev-list command to return only commits that aren't reachable by SHA
		cmd.Add(command.WithArg(fmt.Sprintf("^%s", filter.AfterRef)))
	}
	// add refCommitSHA as starting point
	cmd.Add(command.WithArg(ref))

	if len(filter.Path) != 0 {
		cmd.Add(command.WithPostSepArg(filter.Path))
	}

	// add pagination if requested
	// TODO: we should add absolut limits to protect git (return error)
	if limit > 0 {
		cmd.Add(command.WithFlag("--max-count", strconv.Itoa(limit)))

		if page > 1 {
			cmd.Add(command.WithFlag("--skip", strconv.Itoa((page-1)*limit)))
		}
	}

	if filter.Since > 0 || filter.Until > 0 {
		cmd.Add(command.WithFlag("--date", "unix"))
	}
	if filter.Since > 0 {
		cmd.Add(command.WithFlag("--since", strconv.FormatInt(filter.Since, 10)))
	}
	if filter.Until > 0 {
		cmd.Add(command.WithFlag("--until", strconv.FormatInt(filter.Until, 10)))
	}
	if filter.Committer != "" {
		cmd.Add(command.WithFlag("--committer", filter.Committer))
	}
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		// TODO: handle error in case they don't have a common merge base!
		return nil, processGiteaErrorf(err, "failed to trigger rev-list command")
	}

	return parseLinesToSlice(output.Bytes()), nil
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
func (a Adapter) ListCommits(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	includeStats bool,
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

		if includeStats {
			fileStats, err := getCommitFileStats(ctx, giteaRepo, commit.SHA)
			if err != nil {
				return nil, nil, fmt.Errorf("encountered error getting commit file stats: %w", err)
			}
			commit.FileStats = fileStats
		}

		commits[i] = *commit
	}

	if len(filter.Path) != 0 {
		renameDetailsList, err := getRenameDetails(ctx, giteaRepo, commits, filter.Path)
		if err != nil {
			return nil, nil, err
		}
		cleanedUpCommits := cleanupCommitsForRename(commits, renameDetailsList, filter.Path)
		return cleanedUpCommits, renameDetailsList, nil
	}

	return commits, nil, nil
}

func getCommitFileStats(
	ctx context.Context,
	giteaRepo *gitea.Repository,
	sha string,
) ([]types.CommitFileStats, error) {
	var changeInfoTypes map[string]changeInfoType
	changeInfoTypes, err := getChangeInfoTypes(ctx, giteaRepo, sha)
	if err != nil {
		return nil, fmt.Errorf("failed to get change infos: %w", err)
	}

	changeInfoChanges, err := getChangeInfoChanges(giteaRepo, sha)
	if err != nil {
		return []types.CommitFileStats{}, fmt.Errorf("failed to get change infos: %w", err)
	}

	fileStats := make([]types.CommitFileStats, len(changeInfoChanges))
	i := 0
	for path, info := range changeInfoChanges {
		fileStats[i] = types.CommitFileStats{
			Path:       changeInfoTypes[path].Path,
			OldPath:    changeInfoTypes[path].OldPath,
			Status:     changeInfoTypes[path].Status,
			Insertions: info.Insertions,
			Deletions:  info.Deletions,
		}
		i++
	}

	return fileStats, nil
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
	ctx context.Context,
	giteaRepo *gitea.Repository,
	commits []types.Commit,
	path string,
) ([]types.PathRenameDetails, error) {
	if len(commits) == 0 {
		return []types.PathRenameDetails{}, nil
	}

	renameDetailsList := make([]types.PathRenameDetails, 0, 2)

	renameDetails, err := giteaGetRenameDetails(ctx, giteaRepo, commits[0].SHA, path)
	if err != nil {
		return nil, err
	}
	if renameDetails.Path != "" || renameDetails.OldPath != "" {
		renameDetails.CommitSHABefore = commits[0].SHA
		renameDetailsList = append(renameDetailsList, *renameDetails)
	}

	if len(commits) == 1 {
		return renameDetailsList, nil
	}

	renameDetailsLast, err := giteaGetRenameDetails(ctx, giteaRepo, commits[len(commits)-1].SHA, path)
	if err != nil {
		return nil, err
	}

	if renameDetailsLast.Path != "" || renameDetailsLast.OldPath != "" {
		renameDetailsLast.CommitSHAAfter = commits[len(commits)-1].SHA
		renameDetailsList = append(renameDetailsList, *renameDetailsLast)
	}
	return renameDetailsList, nil
}

func giteaGetRenameDetails(
	ctx context.Context,
	giteaRepo *gitea.Repository,
	ref string,
	path string,
) (*types.PathRenameDetails, error) {
	changeInfos, err := getChangeInfoTypes(ctx, giteaRepo, ref)
	if err != nil {
		return &types.PathRenameDetails{}, fmt.Errorf("failed to get change infos %w", err)
	}

	for _, c := range changeInfos {
		if c.Status == enum.FileDiffStatusRenamed && (c.OldPath == path || c.Path == path) {
			return &types.PathRenameDetails{
				OldPath: c.OldPath,
				Path:    c.Path,
			}, nil
		}
	}

	return &types.PathRenameDetails{}, nil
}

func gitLogNameStatus(giteaRepo *gitea.Repository, ref string) ([]string, error) {
	cmd := command.New("log",
		command.WithFlag("--name-status"),
		command.WithFlag("--format="),
		command.WithFlag("--max-count=1"),
		command.WithArg(ref),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(giteaRepo.Ctx, command.WithDir(giteaRepo.Path), command.WithStdout(output))
	if err != nil {
		return nil, fmt.Errorf("failed to trigger log command: %w", err)
	}
	return parseLinesToSlice(output.Bytes()), nil
}

func gitShowNumstat(giteaRepo *gitea.Repository, ref string) ([]string, error) {
	cmd := command.New("show",
		command.WithFlag("--numstat"),
		command.WithFlag("--format="),
		command.WithArg(ref),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(giteaRepo.Ctx, command.WithDir(giteaRepo.Path), command.WithStdout(output))
	if err != nil {
		return nil, fmt.Errorf("failed to trigger show command: %w", err)
	}
	return parseLinesToSlice(output.Bytes()), nil
}

// Will match "R100\tREADME.md\tREADME_new.md".
// Will extract README.md and README_new.md.
var renameRegex = regexp.MustCompile(`\t(.+)\t(.+)`)

func getChangeInfoTypes(
	ctx context.Context,
	giteaRepo *gitea.Repository,
	ref string,
) (map[string]changeInfoType, error) {
	lines, err := gitLogNameStatus(giteaRepo, ref)
	if err != nil {
		return nil, err
	}

	changeInfoTypes := make(map[string]changeInfoType, len(lines))
	for _, line := range lines {
		c := changeInfoType{}

		matches := renameRegex.FindStringSubmatch(line) // renamed file
		if len(matches) > 0 {
			c.OldPath = matches[1]
			c.Path = matches[2]
		} else {
			lineParts := strings.Split(line, "\t")
			if len(lineParts) != 2 {
				return changeInfoTypes, fmt.Errorf("could not parse file change status string %q", line)
			}
			c.Path = lineParts[1]
		}

		c.Status = convertFileDiffStatus(ctx, line)

		changeInfoTypes[c.Path] = c
	}
	return changeInfoTypes, nil
}

// Will match "31\t0\t.harness/apidiff.yaml".
// Will extract 31, 0 and .harness/apidiff.yaml.
var insertionsDeletionsRegex = regexp.MustCompile(`(\d+)\t(\d+)\t(.+)`)

// Will match "0\t0\tREADME.md => README_new.md".
// Will extract README_new.md.
var renameRegexWithArrow = regexp.MustCompile(`\d+\t\d+\t.+\s=>\s(.+)`)

func getChangeInfoChanges(
	giteaRepo *gitea.Repository,
	ref string,
) (map[string]changeInfoChange, error) {
	lines, err := gitShowNumstat(giteaRepo, ref)
	if err != nil {
		return nil, err
	}

	changeInfos := make(map[string]changeInfoChange, len(lines))
	for _, line := range lines {
		matches := insertionsDeletionsRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			return map[string]changeInfoChange{},
				fmt.Errorf("failed to regex match insertions and deletions for %q", line)
		}

		path := matches[3]
		if renMatches := renameRegexWithArrow.FindStringSubmatch(line); len(renMatches) == 2 {
			path = renMatches[1]
		}

		insertions, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return map[string]changeInfoChange{},
				fmt.Errorf("failed to parse insertions for %q", line)
		}
		deletions, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return map[string]changeInfoChange{},
				fmt.Errorf("failed to parse deletions for %q", line)
		}

		changeInfos[path] = changeInfoChange{
			Insertions: insertions,
			Deletions:  deletions,
		}
	}

	return changeInfos, nil
}

type changeInfoType struct {
	Status  enum.FileDiffStatus
	OldPath string // populated only in case of renames
	Path    string
}
type changeInfoChange struct {
	Insertions int64
	Deletions  int64
}

func convertFileDiffStatus(ctx context.Context, c string) enum.FileDiffStatus {
	switch {
	case strings.HasPrefix(c, "A"):
		return enum.FileDiffStatusAdded
	case strings.HasPrefix(c, "C"):
		return enum.FileDiffStatusCopied
	case strings.HasPrefix(c, "D"):
		return enum.FileDiffStatusDeleted
	case strings.HasPrefix(c, "M"):
		return enum.FileDiffStatusModified
	case strings.HasPrefix(c, "R"):
		return enum.FileDiffStatusRenamed
	default:
		log.Ctx(ctx).Warn().Msgf("encountered unknown change type %s", c)
		return enum.FileDiffStatusUndefined
	}
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

	return GetCommit(ctx, repoPath, rev, "")
}

func (a Adapter) GetFullCommitID(
	ctx context.Context,
	repoPath string,
	shortID string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
	cmd := command.New("rev-parse",
		command.WithArg(shortID),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			return "", errors.NotFound("commit not found %s", shortID)
		}
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
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

// GetCommit returns info about a commit.
// TODO: Move this function outside of the adapter package.
func GetCommit(
	ctx context.Context,
	repoPath string,
	rev string,
	path string,
) (*types.Commit, error) {
	const format = "" +
		fmtCommitHash + fmtZero + // 0
		fmtParentHashes + fmtZero + // 1
		fmtAuthorName + fmtZero + // 2
		fmtAuthorEmail + fmtZero + // 3
		fmtAuthorTime + fmtZero + // 4
		fmtCommitterName + fmtZero + // 5
		fmtCommitterEmail + fmtZero + // 6
		fmtCommitterTime + fmtZero + // 7
		fmtSubject + fmtZero + // 8
		fmtBody // 9

	cmd := command.New("log",
		command.WithFlag("--max-count", "1"),
		command.WithFlag("--format="+format),
		command.WithArg(rev),
	)
	if path != "" {
		cmd.Add(command.WithPostSepArg(path))
	}
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if strings.Contains(err.Error(), "ambiguous argument") {
			return nil, errors.NotFound("revision %q not found", rev)
		}
		return nil, fmt.Errorf("failed to run git to get commit data: %w", err)
	}

	commitLine := output.String()

	if commitLine == "" {
		return nil, errors.InvalidArgument("path %q not found in %s", path, rev)
	}

	const columnCount = 10

	commitData := strings.Split(strings.TrimSpace(commitLine), separatorZero)
	if len(commitData) != columnCount {
		return nil, fmt.Errorf(
			"unexpected git log formatted output, expected %d, but got %d columns", columnCount, len(commitData))
	}

	sha := commitData[0]
	var parentSHAs []string
	if commitData[1] != "" {
		parentSHAs = strings.Split(commitData[1], " ")
	}
	authorName := commitData[2]
	authorEmail := commitData[3]
	authorTimestamp := commitData[4]
	committerName := commitData[5]
	committerEmail := commitData[6]
	committerTimestamp := commitData[7]
	subject := commitData[8]
	body := commitData[9]

	authorTime, _ := time.Parse(time.RFC3339Nano, authorTimestamp)
	committerTime, _ := time.Parse(time.RFC3339Nano, committerTimestamp)

	return &types.Commit{
		SHA:        sha,
		ParentSHAs: parentSHAs,
		Title:      subject,
		Message:    body,
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
