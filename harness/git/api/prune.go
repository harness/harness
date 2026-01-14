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
	"time"

	"github.com/harness/gitness/git/command"
)

type PrunePackedParams struct {
	DryRun bool
	Quiet  bool
}

func (g *Git) PrunePacked(
	ctx context.Context,
	repoPath string,
	params PrunePackedParams,
) error {
	cmd := command.New("prune-packed")

	if params.DryRun {
		cmd.Add(command.WithFlag("--dry-run"))
	}

	if params.Quiet {
		cmd.Add(command.WithFlag("--quiet"))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to prune-packed objects")
	}

	return nil
}

type PruneObjectsParams struct {
	DryRun       bool
	ExpireBefore time.Time
}

func (g *Git) PruneObjects(
	ctx context.Context,
	repoPath string,
	params PruneObjectsParams,
) error {
	cmd := command.New("prune")

	if params.DryRun {
		cmd.Add(command.WithFlag("--dry-run"))
	}

	if !params.ExpireBefore.IsZero() {
		cmd.Add(command.WithFlag(fmt.Sprintf("--expire=%s", params.ExpireBefore.Format(RFC2822DateFormat))))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to prune objects")
	}

	return nil
}
