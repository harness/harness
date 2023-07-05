// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package main

import (
	"github.com/harness/gitness/cli"
	"github.com/harness/gitness/cli/operations/account"
	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/cli/operations/migrate"
	"github.com/harness/gitness/cli/operations/user"
	"github.com/harness/gitness/cli/operations/users"
	"github.com/harness/gitness/cli/server"
	"github.com/harness/gitness/version"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	application = "gitness"
	description = "Gitness Open source edition"
)

func main() {
	args := cli.GetArguments()

	app := kingpin.New(application, description)

	migrate.Register(app)
	server.Register(app, initSystem)

	user.Register(app)
	users.Register(app)

	account.RegisterLogin(app)
	account.RegisterRegister(app)
	account.RegisterLogout(app)

	hooks.Register(app)

	cli.RegisterSwagger(app)

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(args))
}
