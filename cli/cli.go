// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"os"

	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/internal/githook"
)

func GetArguments() []string {
	command := os.Args[0]
	args := os.Args[1:]

	// in case of githooks, translate the arguments coming from git to work with gitness.
	if gitArgs, fromGit := githook.SanitizeArgsForGit(command, args); fromGit {
		return append([]string{hooks.ParamHooks}, gitArgs...)
	}

	return args
}
