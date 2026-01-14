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

package merge

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/rs/zerolog/log"
)

// FindConflicts checks if two git revisions are mergeable and returns list of conflict files if they are not.
func FindConflicts(
	ctx context.Context,
	repoPath,
	base, head string,
) (mergeable bool, treeSHA string, conflicts []string, err error) {
	cmd := command.New("merge-tree",
		command.WithFlag("--write-tree"),
		command.WithFlag("--name-only"),
		command.WithFlag("--no-messages"),
		command.WithFlag("--stdin"))

	stdin := base + " " + head
	stdout := bytes.NewBuffer(nil)

	err = cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdin(strings.NewReader(stdin)),
		command.WithStdout(stdout))

	if err != nil {
		return false, "", nil, errors.Internal(err, "Failed to find conflicts between %s and %s", base, head)
	}

	output := strings.TrimSpace(stdout.String())
	output = strings.TrimSuffix(output, "\000")

	lines := strings.Split(output, "\000")
	if len(lines) < 2 {
		log.Ctx(ctx).Error().Str("output", output).Msg("Unexpected merge-tree output")
		return false, "", nil, errors.Internal(nil,
			"Failed to find conflicts between %s and %s: Unexpected git output", base, head)
	}

	status, err := strconv.Atoi(lines[0])
	if err != nil {
		log.Ctx(ctx).Err(err).Str("output", output).Msg("Unexpected merge status")
		return false, "", nil, errors.Internal(nil,
			"Failed to find conflicts between %s and %s: Unexpected merge status", base, head)
	}

	if status < 0 {
		return false, "", nil, errors.Internal(nil,
			"Failed to find conflicts between %s and %s: Operation blocked. Status=%d", base, head, status)
	}

	treeSHA = lines[1]
	if status == 1 {
		return true, treeSHA, nil, nil // all good, merge possible, no conflicts found
	}

	conflicts = sharedrepo.CleanupMergeConflicts(lines[2:])

	return false, treeSHA, conflicts, nil // conflict found, list of conflicted files returned
}

// CommitCount returns number of commits between the two git revisions.
func CommitCount(
	ctx context.Context,
	repoPath string,
	start, end string,
) (int, error) {
	arg := command.WithArg(end)
	if len(start) > 0 {
		arg = command.WithArg(start + ".." + end)
	}
	cmd := command.New("rev-list", command.WithFlag("--count"), arg)

	stdout := bytes.NewBuffer(nil)

	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(stdout)); err != nil {
		return 0, errors.Internal(err, "failed to rev-list in shared repo")
	}

	commitCount, err := strconv.Atoi(strings.TrimSpace(stdout.String()))
	if err != nil {
		return 0, errors.Internal(err, "failed to parse commit count from rev-list output in shared repo")
	}

	return commitCount, nil
}
