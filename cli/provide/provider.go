// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package provide

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/harness/gitness/cli/session"
	"github.com/harness/gitness/client"

	"github.com/adrg/xdg"
	"github.com/rs/zerolog/log"
)

const DefaultServerURI = "http://localhost:3000"

func Session() session.Session {
	ss, err := newSession()
	if err != nil {
		log.Err(err).Msg("failed to get active session")
		os.Exit(1)
	}

	return ss
}

func Client() client.Client {
	return newClient(Session())
}

func OpenClient(uri string) client.Client {
	return newClient(session.Session{URI: uri})
}

func newSession() (session.Session, error) {
	path, err := xdg.ConfigFile(
		filepath.Join("app", "config.json"),
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return session.Session{URI: DefaultServerURI}, nil
		}
		return session.Session{}, err
	}

	ss, err := session.LoadFromPath(path)
	if err != nil {
		return session.Session{}, err
	}

	if ss.URI == "" {
		ss = ss.SetURI(DefaultServerURI)
	}

	return ss, nil
}

func newClient(ss session.Session) client.Client {
	httpClient := client.NewToken(ss.URI, ss.AccessToken)
	if os.Getenv("DEBUG") == "true" {
		httpClient.SetDebug(true)
	}

	return httpClient
}
