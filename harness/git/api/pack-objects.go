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

type PackObjectsParams struct {
	// Pack unreachable loose objects (and their loose counterparts removed).
	PackLooseUnreachable bool
	// This flag causes an object that is borrowed from an alternate object
	// store to be ignored even if it would have otherwise been packed.
	Local bool
	// This flag causes an object already in a pack to be ignored even if it
	// would have otherwise been packed.
	Incremental bool
	// Only create a packed archive if it would contain at least one object.
	NonEmpty bool
	// Should we show progress
	Quiet bool
}

func (g *Git) PackObjects(
	ctx context.Context,
	repoPath string,
	params PackObjectsParams,
	args ...string,
) error {
	cmd := command.New("pack-objects")

	if params.PackLooseUnreachable {
		cmd.Add(command.WithFlag("--pack-loose-unreachable"))
	}
	if params.Local {
		cmd.Add(command.WithFlag("--local"))
	}
	if params.Incremental {
		cmd.Add(command.WithFlag("--incremental"))
	}
	if params.NonEmpty {
		cmd.Add(command.WithFlag("--non-empty"))
	}
	if params.Quiet {
		cmd.Add(command.WithFlag("--quiet"))
	}

	if len(args) > 0 {
		cmd.Add(command.WithArg(args...))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to pack objects")
	}

	return nil
}
