// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"

	"github.com/harness/gitness/githook"
	gitnessgithook "github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/version"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	application = "gitness-githook"
	description = "A lightweight executable that forwards git server hooks to the gitness API server."
)

func main() {
	// ensure args are properly sanitized if called by git
	command := os.Args[0]
	args := os.Args[1:]
	args, _ = githook.SanitizeArgsForGit(command, args)

	// define new kingpin application and register githooks globally
	app := kingpin.New(application, description)
	app.Version(version.Version.String())
	githook.RegisterAll(app, gitnessgithook.LoadFromEnvironment)

	// trigger execution
	kingpin.MustParse(app.Parse(args))
}
