// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gitea "code.gitea.io/gitea/modules/git"
	gitearef "code.gitea.io/gitea/modules/git/foreachref"
)

const (
	giteaPrettyLogFormat = `--pretty=format:%H`
)

type giteaAdapter struct {
}

func newGiteaAdapter() (giteaAdapter, error) {
	err := gitea.InitSimple(context.Background())
	if err != nil {
		return giteaAdapter{}, err
	}

	return giteaAdapter{}, nil
}

// InitRepository initializes a new Git repository.
func (g giteaAdapter) InitRepository(ctx context.Context, repoPath string, bare bool) error {
	return gitea.InitRepository(ctx, repoPath, bare)
}

// SetDefaultBranch sets the default branch of a repo.
func (g giteaAdapter) SetDefaultBranch(ctx context.Context, repoPath string,
	defaultBranch string, allowEmpty bool) error {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return err
	}
	defer giteaRepo.Close()

	// if requested, error out if branch doesn't exist. Otherwise, blindly set it.
	if !allowEmpty && !giteaRepo.IsBranchExist(defaultBranch) {
		// TODO: ensure this returns not found error to caller
		return fmt.Errorf("branch '%s' does not exist", defaultBranch)
	}

	// change default branch
	err = giteaRepo.SetDefaultBranch(defaultBranch)
	if err != nil {
		return fmt.Errorf("failed to set new default branch: %w", err)
	}

	return nil
}

func (g giteaAdapter) Clone(ctx context.Context, from, to string, opts cloneRepoOptions) error {
	return gitea.Clone(ctx, from, to, gitea.CloneRepoOptions{
		Timeout:       opts.timeout,
		Mirror:        opts.mirror,
		Bare:          opts.bare,
		Quiet:         opts.quiet,
		Branch:        opts.branch,
		Shared:        opts.shared,
		NoCheckout:    opts.noCheckout,
		Depth:         opts.depth,
		Filter:        opts.filter,
		SkipTLSVerify: opts.skipTLSVerify,
	})
}

func (g giteaAdapter) AddFiles(repoPath string, all bool, files ...string) error {
	return gitea.AddChanges(repoPath, all, files...)
}

func (g giteaAdapter) Commit(repoPath string, opts commitChangesOptions) error {
	return gitea.CommitChanges(repoPath, gitea.CommitChangesOptions{
		Committer: &gitea.Signature{
			Name:  opts.committer.identity.name,
			Email: opts.committer.identity.email,
			When:  opts.committer.when,
		},
		Author: &gitea.Signature{
			Name:  opts.author.identity.name,
			Email: opts.author.identity.email,
			When:  opts.author.when,
		},
		Message: opts.message,
	})
}

func (g giteaAdapter) Push(ctx context.Context, repoPath string, opts pushOptions) error {
	return gitea.Push(ctx, repoPath, gitea.PushOptions{
		Remote:  opts.remote,
		Branch:  opts.branch,
		Force:   opts.force,
		Mirror:  opts.mirror,
		Env:     opts.env,
		Timeout: opts.timeout,
	})
}

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) GetTreeNode(ctx context.Context, repoPath string,
	ref string, treePath string) (*treeNode, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	// TODO: handle ErrNotExist :)
	giteaTreeEntry, err := giteaCommit.GetTreeEntryByPath(treePath)
	if err != nil {
		return nil, err
	}

	nodeType, mode, err := mapGiteaNodeToTreeNodeModeAndType(giteaTreeEntry.Mode())
	if err != nil {
		return nil, err
	}

	return &treeNode{
		mode:     mode,
		nodeType: nodeType,
		sha:      giteaTreeEntry.ID.String(),
		name:     giteaTreeEntry.Name(),
		path:     treePath,
	}, nil
}

// GetLatestCommit gets the latest commit of a path relative from the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) GetLatestCommit(ctx context.Context, repoPath string,
	ref string, treePath string) (*commit, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	giteaCommit, err := giteaGetCommitByPath(giteaRepo, ref, treePath)
	if err != nil {
		return nil, fmt.Errorf("error getting latest commit for '%s': %w", treePath, err)
	}

	return mapGiteaCommit(giteaCommit)
}

// giteaGetCommitByPath is a copy of gitea code - required as we want latest commit per specific branch.
func giteaGetCommitByPath(giteaRepo *gitea.Repository, ref string, treePath string) (*gitea.Commit, error) {
	if treePath == "" {
		treePath = "."
	}

	// NOTE: the difference to gitea implementation is passing `ref`.
	stdout, _, runErr := gitea.NewCommand(giteaRepo.Ctx, "log", ref, "-1", giteaPrettyLogFormat, "--", treePath).
		RunStdBytes(&gitea.RunOpts{Dir: giteaRepo.Path})
	if runErr != nil {
		return nil, runErr
	}

	giteaCommits, err := giteaParsePrettyFormatLogToList(giteaRepo, stdout)
	if err != nil {
		return nil, err
	}

	return giteaCommits[0], nil
}

// giteaParsePrettyFormatLogToList is an exact copy of gitea code.
func giteaParsePrettyFormatLogToList(giteaRepo *gitea.Repository, logs []byte) ([]*gitea.Commit, error) {
	var giteaCommits []*gitea.Commit
	if len(logs) == 0 {
		return giteaCommits, nil
	}

	parts := bytes.Split(logs, []byte{'\n'})

	for _, commitID := range parts {
		commit, err := giteaRepo.GetCommit(string(commitID))
		if err != nil {
			return nil, err
		}
		giteaCommits = append(giteaCommits, commit)
	}

	return giteaCommits, nil
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path
// and includes the latest commit for all nodes if requested.
// IMPORTANT: recursive and includeLatestCommit can't be used together.
// Note: ref can be Branch / Tag / CommitSHA.
//
//nolint:gocognit // refactor if needed
func (g giteaAdapter) ListTreeNodes(ctx context.Context, repoPath string,
	ref string, treePath string, recursive bool, includeLatestCommit bool) ([]treeNodeWithCommit, error) {
	if recursive && includeLatestCommit {
		// To avoid potential performance catastrophies, block recursive with includeLatestCommit
		// TODO: this should return bad error to caller if needed?
		// TODO: should this be refactored in two methods?
		return nil, fmt.Errorf("latest commit with recursive query is not supported")
	}

	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	// Get the giteaTree object for the ref
	giteaTree, err := giteaCommit.SubTree(treePath)
	if err != nil {
		return nil, fmt.Errorf("error getting tree for '%s': %w", treePath, err)
	}

	var giteaEntries gitea.Entries
	if recursive {
		giteaEntries, err = giteaTree.ListEntriesRecursive()
	} else {
		giteaEntries, err = giteaTree.ListEntries()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list entries for tree '%s': %w", treePath, err)
	}

	var latestCommits []gitea.CommitInfo
	if includeLatestCommit {
		// TODO: can be speed up with latestCommitCache (currently nil)
		latestCommits, _, err = giteaEntries.GetCommitsInfo(ctx, giteaCommit, treePath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest commits for entries: %w", err)
		}

		if len(latestCommits) != len(giteaEntries) {
			return nil, fmt.Errorf("latest commit info doesn't match tree node info - count differs")
		}
	}

	nodes := make([]treeNodeWithCommit, len(giteaEntries))
	for i := range giteaEntries {
		giteaEntry := giteaEntries[i]

		var nodeType treeNodeType
		var mode treeNodeMode
		nodeType, mode, err = mapGiteaNodeToTreeNodeModeAndType(giteaEntry.Mode())
		if err != nil {
			return nil, err
		}

		// giteaNode.Name() returns the path of the node relative to the tree.
		relPath := giteaEntry.Name()
		name := filepath.Base(relPath)

		var commit *commit
		if includeLatestCommit {
			commit, err = mapGiteaCommit(latestCommits[i].Commit)
			if err != nil {
				return nil, err
			}
		}

		nodes[i] = treeNodeWithCommit{
			treeNode: treeNode{
				nodeType: nodeType,
				mode:     mode,
				sha:      giteaEntry.ID.String(),
				name:     name,
				path:     filepath.Join(treePath, relPath),
			},
			commit: commit,
		}
	}

	return nodes, nil
}

// ListCommits lists the commits reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) ListCommits(ctx context.Context, repoPath string,
	ref string, page int, pageSize int) ([]commit, int64, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, 0, err
	}
	defer giteaRepo.Close()

	// Get the giteaTopCommit object for the ref
	giteaTopCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	giteaCommits, err := giteaTopCommit.CommitsByRange(page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting commits: %w", err)
	}

	totalCount, err := giteaTopCommit.CommitsCount()
	if err != nil {
		return nil, 0, fmt.Errorf("error getting total commit count: %w", err)
	}

	commits := make([]commit, len(giteaCommits))
	for i := range giteaCommits {
		var commit *commit
		commit, err = mapGiteaCommit(giteaCommits[i])
		if err != nil {
			return nil, 0, err
		}
		commits[i] = *commit
	}

	// TODO: save to cast to int from int64, or we expect exceeding int.MaxValue?
	return commits, totalCount, nil
}

// GetCommit returns the (latest) commit for a specific ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) GetCommit(ctx context.Context, repoPath string, ref string) (*commit, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	commit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, err
	}

	return mapGiteaCommit(commit)
}

// GetCommits returns the (latest) commits for a specific list of refs.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) GetCommits(ctx context.Context, repoPath string, refs []string) ([]commit, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	commits := make([]commit, len(refs))
	for i, sha := range refs {
		var giteaCommit *gitea.Commit
		giteaCommit, err = giteaRepo.GetCommit(sha)
		if err != nil {
			return nil, err
		}

		var commit *commit
		commit, err = mapGiteaCommit(giteaCommit)
		if err != nil {
			return nil, err
		}
		commits[i] = *commit
	}

	return commits, nil
}

// GetAnnotatedTag returns the tag for a specific tag sha.
func (g giteaAdapter) GetAnnotatedTag(ctx context.Context, repoPath string, sha string) (*tag, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	return giteaGetAnnotatedTag(giteaRepo, sha)
}

// GetAnnotatedTags returns the tags for a specific list of tag sha.
func (g giteaAdapter) GetAnnotatedTags(ctx context.Context, repoPath string, shas []string) ([]tag, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	tags := make([]tag, len(shas))
	for i, sha := range shas {
		var tag *tag
		tag, err = giteaGetAnnotatedTag(giteaRepo, sha)
		if err != nil {
			return nil, err
		}

		tags[i] = *tag
	}

	return tags, nil
}

// giteaGetAnnotatedTag is a custom implementation to retrieve an annotated tag from a sha.
// The code is following parts of the gitea implementation.
//
// IMPORTANT: This is required as all gitea implementations of form get*Tag
// are having huge performance issues (with 2,500 tags it took seconds per single tag!)
func giteaGetAnnotatedTag(giteaRepo *gitea.Repository, sha string) (*tag, error) {
	// The tag is an annotated tag with a message.
	writer, reader, cancel := giteaRepo.CatFileBatch(giteaRepo.Ctx)
	defer cancel()
	if _, err := writer.Write([]byte(sha + "\n")); err != nil {
		return nil, err
	}
	tagSha, typ, size, err := gitea.ReadBatchLine(reader)
	if err != nil {
		if errors.Is(err, io.EOF) || gitea.IsErrNotExist(err) {
			return nil, fmt.Errorf("tag with sha %s does not exist", sha)
		}
		return nil, err
	}
	if typ != string(gitObjectTypeTag) {
		return nil, fmt.Errorf("git object is of type '%s', expected tag", typ)
	}

	// read the remaining rawData
	rawData, err := io.ReadAll(io.LimitReader(reader, size))
	if err != nil {
		return nil, err
	}
	_, err = reader.Discard(1)
	if err != nil {
		return nil, err
	}

	tag, err := parseTagDataFromCatFile(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tag '%s': %w", sha, err)
	}

	// fill in the sha
	tag.sha = string(tagSha)

	return tag, nil
}

const (
	pgpSignatureBeginToken = "\n-----BEGIN PGP SIGNATURE-----\n" //#nosec G101
	pgpSignatureEndToken   = "\n-----END PGP SIGNATURE-----"     //#nosec G101
)

// parseTagDataFromCatFile parses a tag from a cat-file output.
func parseTagDataFromCatFile(data []byte) (*tag, error) {
	tag := &tag{}
	p := 0
	var err error

	// parse object Id
	tag.targetSha, p, err = giteaParseCatFileLine(data, p, "object")
	if err != nil {
		return nil, err
	}

	// parse object type
	rawType, p, err := giteaParseCatFileLine(data, p, "type")
	if err != nil {
		return nil, err
	}

	tag.targetType, err = parseGitObjectType(rawType)
	if err != nil {
		return nil, err
	}

	// parse tag name
	tag.name, p, err = giteaParseCatFileLine(data, p, "tag")
	if err != nil {
		return nil, err
	}

	// parse tagger
	rawTaggerInfo, p, err := giteaParseCatFileLine(data, p, "tagger")
	if err != nil {
		return nil, err
	}
	tag.tagger, err = parseSignatureFromCatFileLine(rawTaggerInfo)
	if err != nil {
		return nil, err
	}

	// remainder is message and gpg (remove leading and tailing new lines)
	message := string(bytes.Trim(data[p:], "\n"))

	// handle gpg signature
	pgpEnd := strings.Index(message, pgpSignatureEndToken)
	if pgpEnd > -1 {
		messageStart := pgpEnd + len(pgpSignatureEndToken)
		// for now we just remove the signature (and trim any separating new lines)
		// TODO: add support for GPG signature of tags
		message = strings.TrimLeft(message[messageStart:], "\n")
	}

	tag.message = message

	// get title from message
	tag.title = message
	titleEnd := strings.IndexByte(message, '\n')
	if titleEnd > -1 {
		tag.title = message[:titleEnd]
	}

	return tag, nil
}

func giteaParseCatFileLine(data []byte, start int, header string) (string, int, error) {
	// for simplicity only look at data from start onwards
	data = data[start:]

	lenHeader := len(header)
	lenData := len(data)
	if lenData < lenHeader {
		return "", 0, fmt.Errorf("expected '%s' but line only contains '%s'", header, string(data))
	}
	if string(data[:lenHeader]) != header {
		return "", 0, fmt.Errorf("expected '%s' but started with '%s'", header, string(data[:lenHeader]))
	}

	// get end of line and start of next line (used externaly, transpose with provided start index)
	lineEnd := bytes.IndexByte(data, '\n')
	externalNextLine := start + lineEnd + 1
	if lineEnd == -1 {
		lineEnd = lenData
		externalNextLine = start + lenData
	}

	// if there's no data, return an error (have to consider for ' ')
	if lineEnd <= lenHeader+1 {
		return "", 0, fmt.Errorf("no data for line of type '%s'", header)
	}

	return string(data[lenHeader+1 : lineEnd]), externalNextLine, nil
}

// defaultGitTimeLayout is the (default) time format printed by git.
const defaultGitTimeLayout = "Mon Jan _2 15:04:05 2006 -0700"

// parseSignatureFromCatFileLine parses the signature from a cat-file output.
// This is used for commit / tag outputs. Input will be similar to (without 'author 'prefix):
// - author Max Mustermann <mm@gitness.io> 1666401234 -0700
// - author Max Mustermann <mm@gitness.io> Tue Oct 18 05:13:26 2022 +0530
// TODO: method is leaning on gitea code - requires reference?
func parseSignatureFromCatFileLine(line string) (signature, error) {
	sig := signature{}
	emailStart := strings.LastIndexByte(line, '<')
	emailEnd := strings.LastIndexByte(line, '>')
	if emailStart == -1 || emailEnd == -1 || emailEnd < emailStart {
		return signature{}, fmt.Errorf("signature is missing email ('%s')", line)
	}

	// name requires that there is at least one char followed by a space (so emailStart >= 2)
	if emailStart < 2 {
		return signature{}, fmt.Errorf("signature is missing name ('%s')", line)
	}

	sig.identity.name = line[:emailStart-1]
	sig.identity.email = line[emailStart+1 : emailEnd]

	timeStart := emailEnd + 2
	if timeStart >= len(line) {
		return signature{}, fmt.Errorf("signature is missing time ('%s')", line)
	}

	// Check if time format is written date time format (e.g Thu, 07 Apr 2005 22:13:13 +0200)
	// we can check that by ensuring that the date time part starts with a non-digit character.
	if line[timeStart] > '9' {
		var err error
		sig.when, err = time.Parse(defaultGitTimeLayout, line[timeStart:])
		if err != nil {
			return signature{}, fmt.Errorf("failed to time.parse signature time ('%s'): %w", line, err)
		}

		return sig, nil
	}

	// Otherwise we have to manually parse unix time and time zone
	endOfUnixTime := timeStart + strings.IndexByte(line[timeStart:], ' ')
	if endOfUnixTime <= timeStart {
		return signature{}, fmt.Errorf("signature is missing unix time ('%s')", line)
	}

	unixSeconds, err := strconv.ParseInt(line[timeStart:endOfUnixTime], 10, 64)
	if err != nil {
		return signature{}, fmt.Errorf("failed to parse unix time ('%s'): %w", line, err)
	}

	// parse time zone
	startOfTimeZone := endOfUnixTime + 1 // +1 for space
	endOfTimeZone := startOfTimeZone + 5 // +5 for '+0700'
	if startOfTimeZone >= len(line) || endOfTimeZone > len(line) {
		return signature{}, fmt.Errorf("signature is missing time zone ('%s')", line)
	}

	// get and disect timezone, e.g. '+0700'
	rawTimeZone := line[startOfTimeZone:endOfTimeZone]
	rawTimeZoneH := rawTimeZone[1:3]  // gets +[07]00
	rawTimeZoneMin := rawTimeZone[3:] // gets +07[00]
	timeZoneH, err := strconv.ParseInt(rawTimeZoneH, 10, 64)
	if err != nil {
		return signature{}, fmt.Errorf("failed to parse hours of time zone ('%s'): %w", line, err)
	}
	timeZoneMin, err := strconv.ParseInt(rawTimeZoneMin, 10, 64)
	if err != nil {
		return signature{}, fmt.Errorf("failed to parse minutes of time zone ('%s'): %w", line, err)
	}

	timeZoneOffsetInSec := int(timeZoneH*60+timeZoneMin) * 60
	if rawTimeZone[0] == '-' {
		timeZoneOffsetInSec *= -1
	}
	timeZone := time.FixedZone("", timeZoneOffsetInSec)

	// create final time using unix and timezone translation
	sig.when = time.Unix(unixSeconds, 0).In(timeZone)

	return sig, nil
}

func defaultInstructor(_ walkReferencesEntry) (walkInstruction, error) {
	return walkInstructionHandle, nil
}

// WalkReferences uses the provided options to filter the available references of the repo,
// and calls the handle function for every matching node.
// The instructor & handler are called with a map that contains the matching value for every field provided in fields.
// TODO: walkGiteaReferences related code should be moved to separate file.
func (g giteaAdapter) WalkReferences(ctx context.Context,
	repoPath string, handler walkReferencesHandler, opts *walkReferencesOptions) error {
	// backfil optional options
	if opts.instructor == nil {
		opts.instructor = defaultInstructor
	}
	if len(opts.fields) == 0 {
		opts.fields = []gitReferenceField{gitReferenceFieldRefName, gitReferenceFieldObjectName}
	}
	if opts.maxWalkDistance <= 0 {
		opts.maxWalkDistance = math.MaxInt32
	}
	if opts.patterns == nil {
		opts.patterns = []string{}
	}
	if string(opts.sort) == "" {
		opts.sort = gitReferenceFieldRefName
	}

	// prepare for-each-ref input
	sortArg := mapToGiteaReferenceSortingArgument(opts.sort, opts.order)
	rawFields := make([]string, len(opts.fields))
	for i := range opts.fields {
		rawFields[i] = string(opts.fields[i])
	}
	giteaFormat := gitearef.NewFormat(rawFields...)

	// initializer pipeline for output processing
	pipeOut, pipeIn := io.Pipe()
	defer pipeOut.Close()
	defer pipeIn.Close()
	stderr := strings.Builder{}
	rc := &gitea.RunOpts{Dir: repoPath, Stdout: pipeIn, Stderr: &stderr}

	// create sort argument

	go func() {
		// create array for args as patterns have to be passed as separate args.
		args := []string{
			"for-each-ref",
			"--format",
			giteaFormat.Flag(),
			"--sort",
			sortArg,
			"--count",
			fmt.Sprint(opts.maxWalkDistance),
			"--ignore-case",
		}
		args = append(args, opts.patterns...)
		err := gitea.NewCommand(ctx, args...).Run(rc)
		if err != nil {
			_ = pipeIn.CloseWithError(gitea.ConcatenateError(err, stderr.String()))
		} else {
			_ = pipeIn.Close()
		}
	}()

	parser := giteaFormat.Parser(pipeOut)
	return walkGiteaReferenceParser(parser, handler, opts)
}

func walkGiteaReferenceParser(parser *gitearef.Parser, handler walkReferencesHandler,
	opts *walkReferencesOptions) error {
	for i := int32(0); i < opts.maxWalkDistance; i++ {
		// parse next line - nil if end of output reached or an error occurred.
		rawRef := parser.Next()
		if rawRef == nil {
			break
		}

		// convert to correct map.
		ref, err := mapGiteaRawRef(rawRef)
		if err != nil {
			return err
		}

		// check with the instructor on the next instruction.
		instruction, err := opts.instructor(ref)
		if err != nil {
			return fmt.Errorf("error getting instruction: %w", err)
		}

		if instruction == walkInstructionSkip {
			continue
		}
		if instruction == walkInstructionStop {
			break
		}

		// otherwise handle the reference.
		err = handler(ref)
		if err != nil {
			return fmt.Errorf("error handling reference: %w", err)
		}
	}

	if err := parser.Err(); err != nil {
		return fmt.Errorf("failed to parse output: %w", err)
	}

	return nil
}

func mapGiteaRawRef(raw map[string]string) (map[gitReferenceField]string, error) {
	res := make(map[gitReferenceField]string, len(raw))
	for k, v := range raw {
		gitRefField, err := parseGitReferenceField(k)
		if err != nil {
			return nil, err
		}
		res[gitRefField] = v
	}

	return res, nil
}

func mapToGiteaReferenceSortingArgument(s gitReferenceField, o sortOrder) string {
	sortBy := string(gitReferenceFieldRefName)
	desc := o == sortOrderDesc

	if s == gitReferenceFieldCreatorDate {
		sortBy = string(gitReferenceFieldCreatorDate)
		if o == sortOrderDefault {
			desc = true
		}
	}

	if desc {
		return "-" + sortBy
	}

	return sortBy
}

// GetSubmodule returns the submodule at the given path reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g giteaAdapter) GetSubmodule(ctx context.Context, repoPath string,
	ref string, treePath string) (*submodule, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	giteaSubmodule, err := giteaCommit.GetSubModule(treePath)
	if err != nil {
		return nil, fmt.Errorf("error getting submodule '%s' from commit: %w", ref, err)
	}

	return &submodule{
		name: giteaSubmodule.Name,
		url:  giteaSubmodule.URL,
	}, nil
}

// GetBlob returns the blob at the given path reachable from ref.
// Note: sha is the object sha.
func (g giteaAdapter) GetBlob(ctx context.Context, repoPath string, sha string, sizeLimit int64) (*blob, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	giteaBlob, err := giteaRepo.GetBlob(sha)
	if err != nil {
		return nil, fmt.Errorf("error getting blob '%s': %w", sha, err)
	}

	reader, err := giteaBlob.DataAsync()
	if err != nil {
		return nil, fmt.Errorf("error opening data for blob '%s': %w", sha, err)
	}

	returnSize := giteaBlob.Size()
	if sizeLimit > 0 && returnSize > sizeLimit {
		returnSize = sizeLimit
	}

	// TODO: ensure it doesn't fail because buff has exact size of bytes required
	buff := make([]byte, returnSize)
	_, err = io.ReadAtLeast(reader, buff, int(returnSize))
	if err != nil {
		return nil, fmt.Errorf("error reading data from blob '%s': %w", sha, err)
	}

	return &blob{
		size:    giteaBlob.Size(),
		content: buff,
	}, nil
}

func mapGiteaCommit(giteaCommit *gitea.Commit) (*commit, error) {
	if giteaCommit == nil {
		return nil, fmt.Errorf("gitea commit is nil")
	}

	author, err := mapGiteaSignature(giteaCommit.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to map gitea author: %w", err)
	}
	committer, err := mapGiteaSignature(giteaCommit.Committer)
	if err != nil {
		return nil, fmt.Errorf("failed to map gitea commiter: %w", err)
	}
	return &commit{
		sha:   giteaCommit.ID.String(),
		title: giteaCommit.Summary(),
		// remove potential tailing newlines from message
		message:   strings.TrimRight(giteaCommit.Message(), "\n"),
		author:    author,
		committer: committer,
	}, nil
}

func mapGiteaNodeToTreeNodeModeAndType(giteaMode gitea.EntryMode) (treeNodeType, treeNodeMode, error) {
	switch giteaMode {
	case gitea.EntryModeBlob:
		return treeNodeTypeBlob, treeNodeModeFile, nil
	case gitea.EntryModeSymlink:
		return treeNodeTypeBlob, treeNodeModeSymlink, nil
	case gitea.EntryModeExec:
		return treeNodeTypeBlob, treeNodeModeExec, nil
	case gitea.EntryModeCommit:
		return treeNodeTypeCommit, treeNodeModeCommit, nil
	case gitea.EntryModeTree:
		return treeNodeTypeTree, treeNodeModeTree, nil
	default:
		return treeNodeTypeBlob, treeNodeModeFile,
			fmt.Errorf("received unknown tree node mode from gitea: '%s'", giteaMode.String())
	}
}

func mapGiteaSignature(giteaSignature *gitea.Signature) (signature, error) {
	if giteaSignature == nil {
		return signature{}, fmt.Errorf("gitea signature is nil")
	}

	return signature{
		identity: identity{
			name:  giteaSignature.Name,
			email: giteaSignature.Email,
		},
		when: giteaSignature.When,
	}, nil
}
