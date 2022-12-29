// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/harness/gitness/cli/operations/account"
	"github.com/harness/gitness/cli/operations/hooks"
	"github.com/harness/gitness/cli/operations/user"
	"github.com/harness/gitness/cli/operations/users"
	"github.com/harness/gitness/cli/server"
	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/client"
	"github.com/harness/gitness/version"

	"github.com/adrg/xdg"
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

	hooks.Register(app, httpClient)

	registerSwagger(app)

	kingpin.Version(version.Version.String())
	kingpin.MustParse(app.Parse(args))
}

func initialize(ss *session.Session, httpClient *client.HTTPClient) error {
	// todo: refactor asap, we need to get rid of literal value 'server', 'login' & 'register'.
	if len(os.Args) > 1 && os.Args[1] == "server" {
		return nil
	}
	path, err := xdg.ConfigFile(
		filepath.Join("app", "config.json"),
	)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	*ss, err = session.LoadFromPath(path)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			break
		case errors.Is(err, session.ErrTokenExpired) &&
			len(os.Args) == 2 && (os.Args[1] == "login" || os.Args[1] == "register"):
			break
		default:
			return err
		}
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

func getArguments() []string {
	// for git operations, the first argument is "hooks", followed by other relevant arguments
	if os.Args[0] == "hooks/update" || os.Args[0] == "hooks/pre-receive" || os.Args[0] == "hooks/post-receive" {
		return append([]string{"hooks", path.Base(os.Args[0])}, os.Args[1:]...)
	}
	return os.Args[1:]
}
