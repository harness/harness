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

	"github.com/harness/gitness/git/command"
)

type PackRefsParams struct {
	// All option causes all refs to be packed as well,
	// with the exception of hidden refs, broken refs, and symbolic refs.
	// Useful for a repository with many branches of historical interests.
	All bool
	// NoPrune command usually removes loose refs under $GIT_DIR/refs
	// hierarchy after packing them. This option tells it not to.
	NoPrune bool
	// Auto pack refs as needed depending on the current state of the ref database.
	// The behavior depends on the ref format used by the repository and may change in the future.
	Auto bool
	// Include pack refs based on a glob(7) pattern. Repetitions of this option accumulate inclusion patterns.
	// If a ref is both included in --include and --exclude, --exclude takes precedence.
	Include string
	// Exclude doesn't pack refs matching the given glob(7) pattern.
	// Repetitions of this option accumulate exclusion patterns.
	Exclude string
}

func (g *Git) PackRefs(
	ctx context.Context,
	repoPath string,
	params PackRefsParams,
) error {
	cmd := command.New("pack-refs")

	if params.All {
		cmd.Add(command.WithFlag("--all"))
	}
	if params.NoPrune {
		cmd.Add(command.WithFlag("--no-prune"))
	}
	if params.Auto {
		cmd.Add(command.WithFlag("--auto"))
	}
	if params.Include != "" {
		cmd.Add(command.WithFlag("--include", params.Include))
	}
	if params.Exclude != "" {
		cmd.Add(command.WithFlag("--exclude", params.Exclude))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to pack refs")
	}

	return nil
}
