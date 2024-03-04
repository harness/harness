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

package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

// CommitGPGSignature represents a git commit signature part.
type CommitGPGSignature struct {
	Signature string
	Payload   string
}

type CommitChangesOptions struct {
	Committer Signature
	Author    Signature
	Message   string
}

type Commit struct {
	SHA       sha.SHA   `json:"sha"`
	Title     string    `json:"title"`
	Message   string    `json:"message,omitempty"`
	Author    Signature `json:"author"`
	Committer Signature `json:"committer"`
	Signature *CommitGPGSignature
	Parents   []sha.SHA
	FileStats CommitFileStats
}

type CommitFilter struct {
	Path      string
	AfterRef  string
	Since     int64
	Until     int64
	Committer string
}

// CommitDivergenceRequest contains the refs for which the converging commits should be counted.
type CommitDivergenceRequest struct {
	// From is the ref from which the counting of the diverging commits starts.
	From string
	// To is the ref at which the counting of the diverging commits ends.
	To string
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32
}

type PathRenameDetails struct {
	OldPath         string
	NewPath         string
	CommitSHABefore sha.SHA
	CommitSHAAfter  sha.SHA
}

// GetLatestCommit gets the latest commit of a path relative from the provided revision.
func (g *Git) GetLatestCommit(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) (*Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	return getCommit(ctx, repoPath, rev, treePath)
}

func getCommits(
	ctx context.Context,
	repoPath string,
	commitIDs []string,
) ([]*Commit, error) {
	if len(commitIDs) == 0 {
		return nil, nil
	}
	commits := make([]*Commit, 0, len(commitIDs))
	for _, commitID := range commitIDs {
		commit, err := getCommit(ctx, repoPath, commitID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get commit '%s': %w", commitID, err)
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

func (g *Git) listCommitSHAs(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	filter CommitFilter,
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
		return nil, processGitErrorf(err, "failed to trigger rev-list command")
	}

	return parseLinesToSlice(output.Bytes()), nil
}

// ListCommitSHAs lists the commits reachable from ref.
// Note: ref & afterRef can be Branch / Tag / CommitSHA.
// Note: commits returned are [ref->...->afterRef).
func (g *Git) ListCommitSHAs(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	filter CommitFilter,
) ([]string, error) {
	return g.listCommitSHAs(ctx, repoPath, ref, page, limit, filter)
}

// ListCommits lists the commits reachable from ref.
// Note: ref & afterRef can be Branch / Tag / CommitSHA.
// Note: commits returned are [ref->...->afterRef).
func (g *Git) ListCommits(
	ctx context.Context,
	repoPath string,
	ref string,
	page int,
	limit int,
	includeFileStats bool,
	filter CommitFilter,
) ([]*Commit, []PathRenameDetails, error) {
	if repoPath == "" {
		return nil, nil, ErrRepositoryPathEmpty
	}

	commitSHAs, err := g.listCommitSHAs(ctx, repoPath, ref, page, limit, filter)
	if err != nil {
		return nil, nil, err
	}

	commits, err := getCommits(ctx, repoPath, commitSHAs)
	if err != nil {
		return nil, nil, err
	}

	if includeFileStats {
		err = includeFileStatsInCommits(ctx, repoPath, commits)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(filter.Path) != 0 {
		renameDetailsList, err := getRenameDetails(ctx, repoPath, commits, filter.Path)
		if err != nil {
			return nil, nil, err
		}
		cleanedUpCommits := cleanupCommitsForRename(commits, renameDetailsList, filter.Path)
		return cleanedUpCommits, renameDetailsList, nil
	}

	return commits, nil, nil
}

func includeFileStatsInCommits(
	ctx context.Context,
	repoPath string,
	commits []*Commit,
) error {
	for i, commit := range commits {
		fileStats, err := getFileStats(ctx, repoPath, commit.SHA)
		if err != nil {
			return fmt.Errorf("failed to get file stat: %w", err)
		}
		commits[i].FileStats = fileStats
	}
	return nil
}

type CommitFileStats struct {
	Added    []string
	Modified []string
	Removed  []string
}

func getFileStats(
	ctx context.Context,
	repoPath string,
	sha sha.SHA,
) (CommitFileStats, error) {
	changeInfos, err := getChangeInfos(ctx, repoPath, sha.String())
	if err != nil {
		return CommitFileStats{}, fmt.Errorf("failed to get change infos: %w", err)
	}
	fileStats := CommitFileStats{
		Added:    make([]string, 0),
		Removed:  make([]string, 0),
		Modified: make([]string, 0),
	}
	for _, c := range changeInfos {
		switch {
		case c.ChangeType == enum.FileDiffStatusModified || c.ChangeType == enum.FileDiffStatusRenamed:
			fileStats.Modified = append(fileStats.Modified, c.Path)
		case c.ChangeType == enum.FileDiffStatusDeleted:
			fileStats.Removed = append(fileStats.Removed, c.Path)
		case c.ChangeType == enum.FileDiffStatusAdded || c.ChangeType == enum.FileDiffStatusCopied:
			fileStats.Added = append(fileStats.Added, c.Path)
		case c.ChangeType == enum.FileDiffStatusUndefined:
		default:
			log.Ctx(ctx).Warn().Msgf("unknown change type %q for path %q",
				c.ChangeType, c.Path)
		}
	}
	return fileStats, nil
}

// In case of rename of a file, same commit will be listed twice - Once in old file and second time in new file.
// Hence, we are making it a pattern to only list it as part of new file and not as part of old file.
func cleanupCommitsForRename(
	commits []*Commit,
	renameDetails []PathRenameDetails,
	path string,
) []*Commit {
	if len(commits) == 0 {
		return commits
	}
	for _, renameDetail := range renameDetails {
		// Since rename details is present it implies that we have commits and hence don't need null check.
		if commits[0].SHA.Equal(renameDetail.CommitSHABefore) && path == renameDetail.OldPath {
			return commits[1:]
		}
	}
	return commits
}

func getRenameDetails(
	ctx context.Context,
	repoPath string,
	commits []*Commit,
	path string,
) ([]PathRenameDetails, error) {
	if len(commits) == 0 {
		return []PathRenameDetails{}, nil
	}

	renameDetailsList := make([]PathRenameDetails, 0, 2)

	renameDetails, err := gitGetRenameDetails(ctx, repoPath, commits[0].SHA.String(), path)
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

	renameDetailsLast, err := gitGetRenameDetails(ctx, repoPath, commits[len(commits)-1].SHA.String(), path)
	if err != nil {
		return nil, err
	}

	if renameDetailsLast.NewPath != "" || renameDetailsLast.OldPath != "" {
		renameDetailsLast.CommitSHAAfter = commits[len(commits)-1].SHA
		renameDetailsList = append(renameDetailsList, *renameDetailsLast)
	}
	return renameDetailsList, nil
}

func gitGetRenameDetails(
	ctx context.Context,
	repoPath string,
	ref string,
	path string,
) (*PathRenameDetails, error) {
	changeInfos, err := getChangeInfos(ctx, repoPath, ref)
	if err != nil {
		return &PathRenameDetails{}, fmt.Errorf("failed to get change infos %w", err)
	}

	for _, c := range changeInfos {
		if c.ChangeType == enum.FileDiffStatusRenamed && (c.Path == path || c.NewPath == path) {
			return &PathRenameDetails{
				OldPath: c.Path,
				NewPath: c.NewPath,
			}, nil
		}
	}

	return &PathRenameDetails{}, nil
}

func getChangeInfos(
	ctx context.Context,
	repoPath string,
	ref string,
) ([]changeInfo, error) {
	cmd := command.New("log",
		command.WithArg(ref),
		command.WithFlag("--name-status"),
		command.WithFlag("--pretty=format:", "-1"),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		return nil, fmt.Errorf("failed to trigger log command: %w", err)
	}
	lines := parseLinesToSlice(output.Bytes())

	changeInfos, err := getFileChangeTypeFromLog(ctx, lines)
	if err != nil {
		return nil, err
	}
	return changeInfos, nil
}

type changeInfo struct {
	ChangeType enum.FileDiffStatus
	Path       string
	// populated only in case of renames
	NewPath string
}

func getFileChangeTypeFromLog(
	ctx context.Context,
	changeStrings []string,
) ([]changeInfo, error) {
	changeInfos := make([]changeInfo, len(changeStrings))
	for i, changeString := range changeStrings {
		changeStringSplit := strings.Split(changeString, "\t")
		if len(changeStringSplit) < 1 {
			return changeInfos, fmt.Errorf("could not parse changeString %q", changeString)
		}

		c := changeInfo{}
		c.ChangeType = convertChangeType(ctx, changeStringSplit[0])
		c.Path = changeStringSplit[1]
		if len(changeStringSplit) == 3 {
			c.NewPath = changeStringSplit[2]
		}
		changeInfos[i] = c
	}
	return changeInfos, nil
}

func convertChangeType(ctx context.Context, c string) enum.FileDiffStatus {
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
func (g *Git) GetCommit(
	ctx context.Context,
	repoPath string,
	rev string,
) (*Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	return getCommit(ctx, repoPath, rev, "")
}

func (g *Git) GetFullCommitID(
	ctx context.Context,
	repoPath string,
	shortID string,
) (sha.SHA, error) {
	if repoPath == "" {
		return sha.SHA{}, ErrRepositoryPathEmpty
	}
	cmd := command.New("rev-parse",
		command.WithArg(shortID),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			return sha.SHA{}, errors.NotFound("commit not found %s", shortID)
		}
		return sha.SHA{}, err
	}
	return sha.New(output.Bytes())
}

// GetCommits returns the (latest) commits for a specific list of refs.
// Note: ref can be Branch / Tag / CommitSHA.
func (g *Git) GetCommits(
	ctx context.Context,
	repoPath string,
	refs []string,
) ([]*Commit, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	return getCommits(ctx, repoPath, refs)
}

// GetCommitDivergences returns the count of the diverging commits for all branch pairs.
// IMPORTANT: If a max is provided it limits the overal count of diverging commits
// (max 10 could lead to (0, 10) while it's actually (2, 12)).
func (g *Git) GetCommitDivergences(
	ctx context.Context,
	repoPath string,
	requests []CommitDivergenceRequest,
	max int32,
) ([]CommitDivergence, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	var err error
	res := make([]CommitDivergence, len(requests))
	for i, req := range requests {
		res[i], err = g.getCommitDivergence(ctx, repoPath, req, max)
		if errors.IsNotFound(err) {
			res[i] = CommitDivergence{Ahead: -1, Behind: -1}
			continue
		}
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// getCommitDivergence returns the count of diverging commits for a pair of branches.
// IMPORTANT: If a max is provided it limits the overall count of diverging commits
// (max 10 could lead to (0, 10) while it's actually (2, 12)).
func (g *Git) getCommitDivergence(
	ctx context.Context,
	repoPath string,
	req CommitDivergenceRequest,
	max int32,
) (CommitDivergence, error) {
	cmd := command.New("rev-list",
		command.WithFlag("--count"),
		command.WithFlag("--left-right"),
	)
	// limit count if requested.
	if max > 0 {
		cmd.Add(command.WithFlag("--max-count", strconv.Itoa(int(max))))
	}
	// add query to get commits without shared base commits
	cmd.Add(command.WithArg(req.From + "..." + req.To))

	stdout := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(stdout))
	if err != nil {
		return CommitDivergence{},
			processGitErrorf(err, "git rev-list failed for '%s...%s'", req.From, req.To)
	}

	// parse output, e.g.: `1       2\n`
	output := stdout.String()
	rawLeft, rawRight, ok := strings.Cut(output, "\t")
	if !ok {
		return CommitDivergence{}, fmt.Errorf("git rev-list returned unexpected output '%s'", output)
	}

	// trim any unnecessary characters
	rawLeft = strings.TrimRight(rawLeft, " \t")
	rawRight = strings.TrimRight(rawRight, " \t\n")

	// parse numbers
	left, err := strconv.ParseInt(rawLeft, 10, 32)
	if err != nil {
		return CommitDivergence{},
			fmt.Errorf("failed to parse git rev-list output for ahead '%s' (full: '%s')): %w",
				rawLeft, output, err)
	}
	right, err := strconv.ParseInt(rawRight, 10, 32)
	if err != nil {
		return CommitDivergence{},
			fmt.Errorf("failed to parse git rev-list output for behind '%s' (full: '%s')): %w",
				rawRight, output, err)
	}

	return CommitDivergence{
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

// getCommit returns info about a commit.
// TODO: This function is used only for last used cache
func getCommit(
	ctx context.Context,
	repoPath string,
	rev string,
	path string,
) (*Commit, error) {
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

	commitSHA := sha.ForceNew(commitData[0])
	var parentSHAs []sha.SHA
	if commitData[1] != "" {
		for _, parentSHA := range strings.Split(commitData[1], " ") {
			parentSHAs = append(parentSHAs, sha.ForceNew(parentSHA))
		}
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

	return &Commit{
		SHA:     commitSHA,
		Parents: parentSHAs,
		Title:   subject,
		Message: body,
		Author: Signature{
			Identity: Identity{
				Name:  authorName,
				Email: authorEmail,
			},
			When: authorTime,
		},
		Committer: Signature{
			Identity: Identity{
				Name:  committerName,
				Email: committerEmail,
			},
			When: committerTime,
		},
	}, nil
}

// GetCommit returns info about a commit.
// TODO: Move this function outside of the api package.
func GetCommit(
	ctx context.Context,
	repoPath string,
	rev string,
) (*Commit, error) {
	wr, rd, cancel := CatFileBatch(ctx, repoPath)
	defer cancel()

	_, _ = wr.Write([]byte(rev + "\n"))

	return getCommitFromBatchReader(ctx, repoPath, rd, rev)
}

func getCommitFromBatchReader(
	ctx context.Context,
	repoPath string,
	rd *bufio.Reader,
	rev string,
) (*Commit, error) {
	output, err := ReadBatchHeaderLine(rd)
	if err != nil {
		return nil, err
	}

	switch output.Type {
	case "missing":
		return nil, errors.NotFound("sha '%s' not found", output.SHA)
	case "tag":
		// then we need to parse the tag
		// and load the commit
		data, err := io.ReadAll(io.LimitReader(rd, output.Size))
		if err != nil {
			return nil, err
		}
		if _, err = rd.Discard(1); err != nil {
			return nil, err
		}
		tag := parseTagData(data)

		commit, err := GetCommit(ctx, repoPath, tag.TargetSha.String())
		if err != nil {
			return nil, err
		}

		commit.Message = strings.TrimSpace(tag.Message)
		commit.Author = tag.Tagger
		commit.Signature = tag.Signature

		return commit, nil
	case "commit":
		commit, err := CommitFromReader(output.SHA, io.LimitReader(rd, output.Size))
		if err != nil {
			return nil, err
		}
		if _, err = rd.Discard(1); err != nil {
			return nil, err
		}

		return commit, nil
	default:
		log.Debug().Msgf("Unknown object type: %s", output.Type)
		_, err = rd.Discard(int(output.Size) + 1)
		if err != nil {
			return nil, err
		}
		return nil, errors.NotFound("rev '%s' not found", rev)
	}
}

// CommitFromReader will generate a Commit from a provided reader
// We need this to interpret commits from cat-file or cat-file --batch
//
// If used as part of a cat-file --batch stream you need to limit the reader to the correct size.
//
//nolint:gocognit
func CommitFromReader(commitSHA sha.SHA, reader io.Reader) (*Commit, error) {
	commit := &Commit{
		SHA:       commitSHA,
		Author:    Signature{},
		Committer: Signature{},
	}

	payloadSB := new(strings.Builder)
	signatureSB := new(strings.Builder)
	messageSB := new(strings.Builder)
	message := false
	pgpsig := false

	bufReader, ok := reader.(*bufio.Reader)
	if !ok {
		bufReader = bufio.NewReader(reader)
	}

readLoop:
	for {
		line, err := bufReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if message {
					_, _ = messageSB.Write(line)
				}
				_, _ = payloadSB.Write(line)
				break readLoop
			}
			return nil, err
		}
		if pgpsig {
			if len(line) > 0 && line[0] == ' ' {
				_, _ = signatureSB.Write(line[1:])
				continue
			}
			pgpsig = false
		}

		if !message {
			// This is probably not correct but is copied from go-gits interpretation...
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) == 0 {
				message = true
				_, _ = payloadSB.Write(line)
				continue
			}

			split := bytes.SplitN(trimmed, []byte{' '}, 2)
			var data []byte
			if len(split) > 1 {
				data = split[1]
			}

			switch string(split[0]) {
			case "tree":
				_, _ = payloadSB.Write(line)
			case "parent":
				commit.Parents = append(commit.Parents, sha.ForceNew(string(data)))
				_, _ = payloadSB.Write(line)
			case "author":
				commit.Author = Signature{}
				commit.Author.Decode(data)
				_, _ = payloadSB.Write(line)
			case "committer":
				commit.Committer = Signature{}
				commit.Committer.Decode(data)
				_, _ = payloadSB.Write(line)
			case "gpgsig":
				_, _ = signatureSB.Write(data)
				_ = signatureSB.WriteByte('\n')
				pgpsig = true
			}
		} else {
			_, _ = messageSB.Write(line)
			_, _ = payloadSB.Write(line)
		}
	}
	commit.Message = messageSB.String()
	commit.Signature = &CommitGPGSignature{
		Signature: signatureSB.String(),
		Payload:   payloadSB.String(),
	}
	if len(commit.Signature.Signature) == 0 {
		commit.Signature = nil
	}

	return commit, nil
}
