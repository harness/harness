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

// ConnectorAuth represents the authentication configuration for a connector.
type ConnectorAuth struct {
	AuthType enum.ConnectorAuthType `json:"type"`
	Basic    *BasicAuthCreds        `json:"basic,omitempty"`
	Bearer   *BearerTokenCreds      `json:"bearer,omitempty"`
}

// BasicAuthCreds represents credentials for basic authentication.
type BasicAuthCreds struct {
	Username string    `json:"username"`
	Password SecretRef `json:"password"`
}

type BearerTokenCreds struct {
	Token SecretRef `json:"token"`
}

func (c *ConnectorAuth) Validate() error {
	switch c.AuthType {
	case enum.ConnectorAuthTypeBasic:
		if c.Basic == nil {
			return fmt.Errorf("basic auth credentials are required")
		}
		if c.Basic.Username == "" || c.Basic.Password.Identifier == "" {
			return fmt.Errorf("basic auth credentials are required")
		}
	case enum.ConnectorAuthTypeBearer:
		if c.Bearer == nil {
			return fmt.Errorf("bearer token credentials are required")
		}
		if c.Bearer.Token.Identifier == "" {
			return fmt.Errorf("bearer token is required")
		}
	default:
		return fmt.Errorf("unsupported auth type: %s", c.AuthType)
	}
	return nil
}
