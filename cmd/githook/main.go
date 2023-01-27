// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package main

import (
	"os"

	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/version"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	application = "githook"
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
	githook.RegisterAll(app)

	// trigger execution
	kingpin.MustParse(app.Parse(args))
}
