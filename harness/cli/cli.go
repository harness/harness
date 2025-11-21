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

package cli

import (
	"os"

	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/git/hook"
)

func GetArguments() []string {
	command := os.Args[0]
	args := os.Args[1:]

	// in case of githooks, translate the arguments coming from git to work with gitness.
	if gitArgs, fromGit := hook.SanitizeArgsForGit(command, args); fromGit {
		return append([]string{hooks.ParamHooks}, gitArgs...)
	}

	return args
}
