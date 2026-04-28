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

package importer

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	connectorRefAccountPrefix = "account."
	connectorRefOrgPrefix     = "org."
)

type ConnectorDef struct {
	Path       string `json:"path"`
	Identifier string `json:"identifier"`
}

type AccessInfo struct {
	Username string
	Password string
	URL      string
}

// DecodeConnectorRef splits a scoped ref into (connector space path, identifier)
// using parentSpacePath as the source of account/org/project segments:
//
//	"account.<id>" -> (<account>,              <id>)
//	"org.<id>"     -> (<account>/<org>,        <id>)
//	"<id>"         -> (<account>/<org>/<proj>, <id>)
func DecodeConnectorRef(parentSpacePath, ref string) (connectorPath, connectorIdentifier string) {
	parts := strings.SplitN(parentSpacePath, "/", 3)

	switch {
	case strings.HasPrefix(ref, connectorRefAccountPrefix):
		return firstNJoined(parts, 1), ref[len(connectorRefAccountPrefix):]
	case strings.HasPrefix(ref, connectorRefOrgPrefix):
		return firstNJoined(parts, 2), ref[len(connectorRefOrgPrefix):]
	default:
		return parentSpacePath, ref
	}
}

func firstNJoined(parts []string, n int) string {
	if n > len(parts) {
		n = len(parts)
	}
	return strings.Join(parts[:n], "/")
}

func (info AccessInfo) URLWithCredentials() (string, error) {
	repoURL, err := url.Parse(info.URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository clone url, %q: %w", info.URL, err)
	}

	repoURL.User = url.UserPassword(info.Username, info.Password)
	cloneURLWithAuth := repoURL.String()

	return cloneURLWithAuth, nil
}
