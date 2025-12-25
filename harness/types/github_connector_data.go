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

package types

import (
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type GithubConnectorData struct {
	APIURL   string         `json:"api_url"`
	Insecure bool           `json:"insecure"`
	Auth     *ConnectorAuth `json:"auth"`
}

func (g *GithubConnectorData) Validate() error {
	if g.Auth == nil {
		return fmt.Errorf("auth is required for github connectors")
	}
	if g.Auth.AuthType != enum.ConnectorAuthTypeBearer {
		return fmt.Errorf("only bearer token auth is supported for github connectors")
	}
	if err := g.Auth.Validate(); err != nil {
		return fmt.Errorf("invalid auth credentials: %w", err)
	}
	return nil
}

func (g *GithubConnectorData) Type() enum.ConnectorType {
	return enum.ConnectorTypeGithub
}
