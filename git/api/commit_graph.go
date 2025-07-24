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
	"context"
	"fmt"

	"github.com/harness/gitness/git/command"
)

// documentation: https://git-scm.com/docs/git-commit-graph

type CommitGraphAction string

const (
	// CommitGraphActionWrite writes a commit-graph file based on the commits found in packfiles.
	CommitGraphActionWrite CommitGraphAction = "write"
	// CommitGraphActionVerify reads the commit-graph file and verify its contents against the object database.
	// Used to check for corrupted data.
	CommitGraphActionVerify CommitGraphAction = "verify"
)

func (a CommitGraphAction) Validate() error {
	switch a {
	case CommitGraphActionWrite, CommitGraphActionVerify:
		return nil
	default:
		return fmt.Errorf("unknown commit graph action: %s", a)
	}
}

func (a CommitGraphAction) String() (string, error) {
	switch a {
	case CommitGraphActionWrite:
		return "write", nil
	case CommitGraphActionVerify:
		return "verify", nil
	default:
		return "", fmt.Errorf("unknown commit graph action: %v", a)
	}
}

type CommitGraphSplitOption string

const (
	// CommitGraphSplitOptionEmpty doesn't exist in git, it is just value
	// to set --split without value.
	CommitGraphSplitOptionEmpty CommitGraphSplitOption = ""
	// CommitGraphSplitOptionReplace overwrites the existing chain with a new one.
	CommitGraphSplitOptionReplace CommitGraphSplitOption = "replace"
)

func (o CommitGraphSplitOption) Validate() error {
	switch o {
	case CommitGraphSplitOptionEmpty, CommitGraphSplitOptionReplace:
		return nil
	default:
		return fmt.Errorf("unknown commit graph split option: %s", o)
	}
}

type CommitGraphParams struct {
	// Action represents command in git cli, can be write or verify.
	Action CommitGraphAction
	// Reachable option generates the new commit graph by walking commits starting at all refs.
	Reachable bool
	// ChangedPaths option computes and write information about the paths changed
	// between a commit and its first parent. This operation can take a while on large repositories.
	// It provides significant performance gains for getting history of a directory or
	// a file with git log -- <path>.
	ChangedPaths bool
	// SizeMultiple
	SizeMultiple int
	// Split option writes the commit-graph as a chain of multiple commit-graph files
	// stored in <dir>/info/commit-graphs.
	Split *CommitGraphSplitOption
}

func (p CommitGraphParams) Validate() error {
	if err := p.Action.Validate(); err != nil {
		return err
	}
	if p.Split != nil {
		if err := p.Split.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (g *Git) CommitGraph(
	ctx context.Context,
	repoPath string,
	params CommitGraphParams,
) error {
	if err := params.Validate(); err != nil {
		return err
	}

	cmd := command.New("commit-graph")

	action, err := params.Action.String()
	if err != nil {
		return err
	}

	cmd.Add(command.WithAction(action))

	if params.Reachable {
		cmd.Add(command.WithFlag("--reachable"))
	}
	if params.ChangedPaths {
		cmd.Add(command.WithFlag("--changed-paths"))
	}
	if params.SizeMultiple > 0 {
		cmd.Add(command.WithFlag(fmt.Sprintf("--size-multiple=%d", params.SizeMultiple)))
	}
	if params.Split != nil {
		if *params.Split == "" {
			cmd.Add(command.WithFlag("--split"))
		} else {
			cmd.Add(command.WithFlag(fmt.Sprintf("--split=%s", *params.Split)))
		}
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to write commit graph")
	}
	return nil
}
