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

type GCParams struct {
	// Aggressive aggressively optimize the repository at the expense of cpu and time of execution.
	Aggressive bool
	// Auto option runs git gc checks whether any housekeeping is required, if not,
	// it exits without performing any work.
	Auto bool
	// Cruft option is  when expiring unreachable objects pack them separately into a cruft pack
	// instead of storing them as loose objects. --cruft is on by default.
	Cruft *bool
	// MaxCruftSize limit the size of new cruft packs to be at most <n> bytes.
	MaxCruftSize uint64
	// Prune prunes loose objects older than date (default is 2 weeks ago,
	// overridable by the config variable gc.pruneExpire). We should never use prune=now!
	Prune any

	// KeepLargestPack all packs except the largest non-cruft pack, any packs marked with a .keep file,
	// and any cruft pack(s) are consolidated into a single pack.
	// When this option is used, gc.bigPackThreshold is ignored.
	KeepLargestPack bool
}

// GC runs git gc command to collect garbage and optimize repository structure.
func (g *Git) GC(
	ctx context.Context,
	repoPath string,
	params GCParams,
) error {
	cmd := command.New("gc")

	if params.Aggressive {
		cmd.Add(command.WithFlag("--aggressive"))
	}

	if params.Auto {
		cmd.Add(command.WithFlag("--auto"))
	}

	if params.Cruft != nil {
		if *params.Cruft {
			cmd.Add(command.WithFlag("--cruft"))
		} else {
			cmd.Add(command.WithFlag("--no-cruft"))
		}
	}

	if params.MaxCruftSize != 0 {
		cmd.Add(command.WithFlag(fmt.Sprintf("--max-cruft-size=%d", params.MaxCruftSize)))
	}

	// prune is by default on
	if params.Prune != nil {
		switch value := params.Prune.(type) {
		case bool:
			if !value {
				cmd.Add(command.WithFlag("--no-prune"))
			}
		case string:
			cmd.Add(command.WithFlag(fmt.Sprintf("--prune=%s", value)))
		case time.Time:
			if !value.IsZero() {
				cmd.Add(command.WithFlag(fmt.Sprintf("--prune=%s", value.Format(RFC2822DateFormat))))
			}
		default:
			return fmt.Errorf("invalid prune value: %v", value)
		}
	}

	if params.KeepLargestPack {
		cmd.Add(command.WithFlag("--keep-largest-pack"))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to run gc command")
	}

	return nil
}
