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

func NewSession() session.Session {
	ss, err := newSession()
	if err != nil {
		log.Err(err).Msg("failed to get active session")
		os.Exit(1)
	}

	return ss
}

func Session() session.Session {
	ss, err := loadSession()
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

func sessionPath() (string, error) {
	return xdg.ConfigFile(filepath.Join("app", "config.json"))
}

func newSession() (session.Session, error) {
	path, err := sessionPath()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return session.Session{}, err
	}

	return session.New(path).SetURI(DefaultServerURI), nil
}

func loadSession() (session.Session, error) {
	path, err := sessionPath()
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
