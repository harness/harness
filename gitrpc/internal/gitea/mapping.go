// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"fmt"
	"strings"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

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
		Sha:   giteaCommit.ID.String(),
		Title: giteaCommit.Summary(),
		// remove potential tailing newlines from message
		Message:   strings.TrimRight(giteaCommit.Message(), "\n"),
		Author:    author,
		Committer: committer,
	}, nil
}

func mapGiteaNodeToTreeNodeModeAndType(giteaMode gitea.EntryMode) (types.TreeNodeType, types.TreeNodeMode, error) {
	switch giteaMode {
	case gitea.EntryModeBlob:
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile, nil
	case gitea.EntryModeSymlink:
		return types.TreeNodeTypeBlob, types.TreeNodeModeSymlink, nil
	case gitea.EntryModeExec:
		return types.TreeNodeTypeBlob, types.TreeNodeModeExec, nil
	case gitea.EntryModeCommit:
		return types.TreeNodeTypeCommit, types.TreeNodeModeCommit, nil
	case gitea.EntryModeTree:
		return types.TreeNodeTypeTree, types.TreeNodeModeTree, nil
	default:
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile,
			fmt.Errorf("received unknown tree node mode from gitea: '%s'", giteaMode.String())
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
