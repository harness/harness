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

package gitea

import (
	"errors"
	"fmt"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/rs/zerolog/log"
)

// Logs the error and message, returns either the provided message or a git equivalent if possible.
// Always logs the full message with error as warning.
func processGiteaErrorf(err error, format string, args ...interface{}) error {
	// create fallback error returned if we can't map it
	fallbackErr := fmt.Errorf(format, args...)

	// always log internal error together with message.
	log.Warn().Msgf("%v: [GITEA] %v", fallbackErr, err)

	// check if it's a RunStdError error (contains raw git error)
	var runStdErr gitea.RunStdError
	if errors.As(err, &runStdErr) {
		return mapGiteaRunStdError(runStdErr, fallbackErr)
	}

	switch {
	// gitea is using errors.New(no such file or directory") exclusively for OpenRepository ... (at least as of now)
	case err.Error() == "no such file or directory":
		return fmt.Errorf("repository not found: %w", types.ErrNotFound)
	case gitea.IsErrNotExist(err):
		return types.ErrNotFound
	case gitea.IsErrBranchNotExist(err):
		return types.ErrNotFound
	default:
		return fallbackErr
	}
}

// TODO: Improve gitea error handling.
// Doubt this will work for all std errors, as git doesn't seem to have nice error codes.
func mapGiteaRunStdError(err gitea.RunStdError, fallback error) error {
	switch {
	// exit status 128 - fatal: A branch named 'mybranch' already exists.
	// exit status 128 - fatal: cannot lock ref 'refs/heads/a': 'refs/heads/a/b' exists; cannot create 'refs/heads/a'
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "exists"):
		return types.ErrAlreadyExists

	// exit status 128 - fatal: 'a/bc/d/' is not a valid branch name.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "not a valid"):
		return types.ErrInvalidArgument

	// exit status 1 - error: branch 'mybranch' not found.
	case err.IsExitCode(1) && strings.Contains(err.Stderr(), "not found"):
		return types.ErrNotFound

	// exit status 128 - fatal: ambiguous argument 'branch1...branch2': unknown revision or path not in the working tree.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "unknown revision"):
		msg := "unknown revision or path not in the working tree"
		// parse the error response from git output
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 0 {
			cols := strings.Split(lines[0], ": ")
			if len(cols) >= 2 {
				msg = cols[1] + ", " + cols[2]
			}
		}
		return fmt.Errorf("%v err: %w", msg, types.ErrNotFound)

	// exit status 128 - fatal: couldn't find remote ref v1.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "couldn't find"):
		return types.ErrNotFound

	// exit status 128 - fatal: unable to access 'http://127.0.0.1:4101/hvfl1xj5fojwlrw77xjflw80uxjous254jrr967rvj/':
	//   Failed to connect to 127.0.0.1 port 4101 after 4 ms: Connection refused
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "Failed to connect"):
		return types.ErrFailedToConnect

	default:
		return fallback
	}
}

func mapGiteaRawRef(raw map[string]string) (map[types.GitReferenceField]string, error) {
	res := make(map[types.GitReferenceField]string, len(raw))
	for k, v := range raw {
		gitRefField, err := types.ParseGitReferenceField(k)
		if err != nil {
			return nil, err
		}
		res[gitRefField] = v
	}

	return res, nil
}

func mapToGiteaReferenceSortingArgument(s types.GitReferenceField, o types.SortOrder) string {
	sortBy := string(types.GitReferenceFieldRefName)
	desc := o == types.SortOrderDesc

	if s == types.GitReferenceFieldCreatorDate {
		sortBy = string(types.GitReferenceFieldCreatorDate)
		if o == types.SortOrderDefault {
			desc = true
		}
	}

	if desc {
		return "-" + sortBy
	}

	return sortBy
}

func mapGiteaCommit(giteaCommit *gitea.Commit) (*types.Commit, error) {
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
	return &types.Commit{
		SHA:   giteaCommit.ID.String(),
		Title: giteaCommit.Summary(),
		// remove potential tailing newlines from message
		Message:   strings.TrimRight(giteaCommit.Message(), "\n"),
		Author:    author,
		Committer: committer,
	}, nil
}

func mapGogitNodeToTreeNodeModeAndType(gogitMode gogitfilemode.FileMode) (types.TreeNodeType, types.TreeNodeMode, error) {
	switch gogitMode {
	case gogitfilemode.Regular, gogitfilemode.Deprecated:
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile, nil
	case gogitfilemode.Symlink:
		return types.TreeNodeTypeBlob, types.TreeNodeModeSymlink, nil
	case gogitfilemode.Executable:
		return types.TreeNodeTypeBlob, types.TreeNodeModeExec, nil
	case gogitfilemode.Submodule:
		return types.TreeNodeTypeCommit, types.TreeNodeModeCommit, nil
	case gogitfilemode.Dir:
		return types.TreeNodeTypeTree, types.TreeNodeModeTree, nil
	default:
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile,
			fmt.Errorf("received unknown tree node mode from gogit: '%s'", gogitMode.String())
	}
}

func mapGiteaSignature(giteaSignature *gitea.Signature) (types.Signature, error) {
	if giteaSignature == nil {
		return types.Signature{}, fmt.Errorf("gitea signature is nil")
	}

	return types.Signature{
		Identity: types.Identity{
			Name:  giteaSignature.Name,
			Email: giteaSignature.Email,
		},
		When: giteaSignature.When,
	}, nil
}
