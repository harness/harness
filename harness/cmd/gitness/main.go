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

package main

import (
	"github.com/harness/gitness/app/api/openapi"
	"github.com/harness/gitness/cli"
	"github.com/harness/gitness/cli/operations/account"
	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/cli/operations/migrate"
	"github.com/harness/gitness/cli/operations/server"
	"github.com/harness/gitness/cli/operations/swagger"
	"github.com/harness/gitness/cli/operations/user"
	"github.com/harness/gitness/cli/operations/users"
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

	swagger.Register(app, openapi.NewOpenAPIService())

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(args))
}
