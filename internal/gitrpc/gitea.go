// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	gitea "code.gitea.io/gitea/modules/git"
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

func (g giteaAdapter) Clone(ctx context.Context, from, to string, opts cloneRepoOption) error {
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
//nolint:gocognit,goimports // refactor if needed
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

// ListBranches lists the branches of the repo.
func (g giteaAdapter) ListBranches(ctx context.Context, repoPath string,
	includeCommit bool, page int, pageSize int) ([]branch, int64, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, 0, err
	}
	defer giteaRepo.Close()

	// first page is always 1 (anything lower is treated as page 1)
	if page < 1 {
		page = 1
	}

	giteaBranches, totalCount, err := giteaRepo.GetBranches((page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting branches: %w", err)
	}

	branches := make([]branch, len(giteaBranches))
	for i := range giteaBranches {
		var branch *branch
		branch, err = mapGiteaBranch(giteaBranches[i], includeCommit)
		if err != nil {
			return nil, 0, err
		}
		branches[i] = *branch
	}

	// TODO: return int instead?
	return branches, int64(totalCount), nil
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

func mapGiteaBranch(giteaBranch *gitea.Branch, includeCommit bool) (*branch, error) {
	if giteaBranch == nil {
		return nil, fmt.Errorf("gitea branch is nil")
	}

	var commit *commit
	if includeCommit {
		giteaCommit, err := giteaBranch.GetCommit()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit of gitea branch")
		}
		commit, err = mapGiteaCommit(giteaCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to map gitea commit: %w", err)
		}
	}

	return &branch{
		name:   giteaBranch.Name,
		commit: commit,
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
		sha:       giteaCommit.ID.String(),
		title:     giteaCommit.Summary(),
		message:   giteaCommit.Message(),
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
