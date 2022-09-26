// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"os"

	"github.com/harness/gitness/cli/operations/account"
	"github.com/harness/gitness/cli/operations/user"
	"github.com/harness/gitness/cli/operations/users"
	"github.com/harness/gitness/cli/server"
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
	app := kingpin.New(application, description)
	server.Register(app)
	user.Register(app)
	users.Register(app)
	account.RegisterLogin(app)
	account.RegisterLogout(app)
	account.RegisterRegister(app)
	registerSwagger(app)

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
