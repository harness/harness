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

// ConnectorConfig is a list of all the connector and their associated config.
type ConnectorConfig struct {
	Github *GithubConnectorData `json:"github,omitempty"`
}

func (c ConnectorConfig) Validate(typ enum.ConnectorType) error {
	switch typ {
	case enum.ConnectorTypeGithub:
		if c.Github != nil {
			return c.Github.Validate()
		}
		return fmt.Errorf("github connector config is required")
	default:
		return fmt.Errorf("connector type %s is not supported", typ)
	}
}
