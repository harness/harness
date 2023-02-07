// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"os"

	"github.com/harness/gitness/cli/operations/account"
	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/cli/operations/migrate"
	"github.com/harness/gitness/cli/operations/user"
	"github.com/harness/gitness/cli/operations/users"
	"github.com/harness/gitness/cli/server"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/version"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	application = "gitness"
	description = "description goes here" // TODO edit this application description
)

// Command parses the command line arguments and then executes a
// subcommand program.
func Command() {
	args := getArguments()

	app := kingpin.New(application, description)

	migrate.Register(app)
	server.Register(app)

	user.Register(app)
	users.Register(app)

	account.RegisterLogin(app)
	account.RegisterRegister(app)
	account.RegisterLogout(app)

	hooks.Register(app)

	registerSwagger(app)

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(args))
}

func getArguments() []string {
	command := os.Args[0]
	args := os.Args[1:]

	// in case of githooks, translate the arguments comming from git to work with gitness.
	if gitArgs, fromGit := githook.SanitizeArgsForGit(command, args); fromGit {
		return append([]string{hooks.ParamHooks}, gitArgs...)
	}

	return args
}
