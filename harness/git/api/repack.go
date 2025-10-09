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

type RepackParams struct {
	// SinglePack packs everything referenced into a single pack.
	// git repack -a
	SinglePack bool
	// RemoveRedundantObjects after packing, if the newly created packs make some existing packs redundant,
	// remove the redundant packs. git repack -d
	RemoveRedundantObjects bool
	// This flag causes an object that is borrowed from an alternate object store to be
	// ignored even if it would have otherwise been packed. git repack -l pass --local to
	// git pack-objects --local.
	Local bool
	// Arrange resulting pack structure so that each successive pack contains at least
	// <factor> times the number of objects as the next-largest pack.
	Geometric int
	// When used with SinglePack and RemoveRedundantObjects, any unreachable objects from existing packs will be appended
	// to the end of the packfile instead of being removed. In addition, any unreachable
	// loose objects will be packed (and their loose counterparts removed).
	KeepUnreachable bool
	// Any unreachable objects are packed into a separate cruft pack
	Cruft bool
	// Expire unreachable objects older than <approxidate> immediately instead of waiting
	// for the next git gc invocation. Only useful with RemoveRedundantObjects
	CruftExpireBefore time.Time
	// Include objects in .keep files when repacking. Note that we still do not delete .keep packs
	// after pack-objects finishes. This means that we may duplicate objects, but this makes the
	// option safe to use when there are concurrent pushes or fetches. This option is generally
	// only useful if you are writing bitmaps with -b or repack.writeBitmaps, as it ensures that
	// the bitmapped packfile has the necessary objects.
	PackKeptObjects bool
	// Write a reachability bitmap index as part of the repack.
	// This only makes sense when used with SinglePack
	// as the bitmaps must be able to refer to all reachable objects
	WriteBitmap bool
	// Write a multi-pack index (see git-multi-pack-index[1]) containing the non-redundant packs
	// git repack --write-midx
	WriteMidx bool
}

func (g *Git) RepackObjects(
	ctx context.Context,
	repoPath string,
	params RepackParams,
) error {
	cmd := command.New("repack")

	if params.SinglePack {
		cmd.Add(command.WithFlag("-a"))
	}
	if params.RemoveRedundantObjects {
		cmd.Add(command.WithFlag("-d"))
	}
	if params.Local {
		cmd.Add(command.WithFlag("-l"))
	}
	if params.Geometric > 0 {
		cmd.Add(command.WithFlag(fmt.Sprintf("--geometric=%d", params.Geometric)))
	}
	if params.KeepUnreachable {
		cmd.Add(command.WithFlag("--keep-unreachable"))
	}
	if params.Cruft {
		cmd.Add(command.WithFlag("--cruft"))
	}
	if !params.CruftExpireBefore.IsZero() {
		cmd.Add(command.WithFlag(fmt.Sprintf("--cruft-expiration=%s",
			params.CruftExpireBefore.Format(RFC2822DateFormat))))
	}
	if params.PackKeptObjects {
		cmd.Add(command.WithFlag("--pack-kept-objects"))
	}
	if params.WriteBitmap {
		cmd.Add(command.WithFlag("--write-bitmap-index"))
	}
	if params.WriteMidx {
		cmd.Add(command.WithFlag("--write-midx"))
	}

	if err := cmd.Run(ctx, command.WithDir(repoPath)); err != nil {
		return processGitErrorf(err, "failed to repack objects")
	}
	return nil
}
