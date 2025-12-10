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

package sharedrepo

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"
)

var reNotOurRef = regexp.MustCompile("upload-pack: not our ref ([a-fA-f0-9]+)$")

// FetchObjects pull git objects from a different repository.
// It doesn't update any references.
func (r *SharedRepo) FetchObjects(
	ctx context.Context,
	source string,
	objectSHAs []sha.SHA,
) error {
	cmd := command.New("fetch",
		command.WithConfig("advice.fetchShowForcedUpdates", "false"),
		command.WithConfig("credential.helper", ""),
		command.WithFlag(
			"--quiet",
			"--no-auto-gc", // because we're fetching objects that are not referenced
			"--no-tags",
			"--no-write-fetch-head",
			"--no-show-forced-updates",
		),
		command.WithArg(source),
	)

	for _, objectSHA := range objectSHAs {
		cmd.Add(command.WithArg(objectSHA.String()))
	}

	err := cmd.Run(ctx, command.WithDir(r.repoPath))
	if err != nil {
		if parts := reNotOurRef.FindStringSubmatch(strings.TrimSpace(err.Error())); parts != nil {
			return errors.InvalidArgumentf("Unrecognized git object: %s", parts[1])
		}
		return fmt.Errorf("failed to fetch objects: %w", err)
	}

	return nil
}
