// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/client"

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
	s := &session.Session{}
	httpClient := &client.HTTPClient{}

	err := initialize(s, httpClient)
	if err != nil {
		panic(err)
	}

	app := kingpin.New(application, description)
	server.Register(app)

	user.Register(app, httpClient)
	users.Register(app, httpClient)

	account.RegisterLogin(app, httpClient, s)
	account.RegisterRegister(app, httpClient, s)
	account.RegisterLogout(app, s)

	registerSwagger(app)

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func initialize(ss *session.Session, httpClient *client.HTTPClient) error {
	path, err := xdg.ConfigFile(
		filepath.Join("app", "config.json"),
	)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	*ss, err = session.LoadFromPath(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if ss.URI == "" {
		// session is immutable
		*ss = ss.SetURI("http://localhost:3000")
	}

	*httpClient = *client.NewToken(ss.URI, ss.AccessToken)
	if os.Getenv("DEBUG") == "true" {
		httpClient.SetDebug(true)
	}

	return nil
}
