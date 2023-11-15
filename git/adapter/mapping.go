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
	"fmt"
	"strings"

	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
)

func mapGiteaRawRef(
	raw map[string]string,
) (map[types.GitReferenceField]string, error) {
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

func mapToGiteaReferenceSortingArgument(
	s types.GitReferenceField,
	o types.SortOrder,
) string {
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

func mapGogitNodeToTreeNodeModeAndType(
	gogitMode gogitfilemode.FileMode,
) (types.TreeNodeType, types.TreeNodeMode, error) {
	//nolint:exhaustive
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

func mapGiteaSignature(
	giteaSignature *gitea.Signature,
) (types.Signature, error) {
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
