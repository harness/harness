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

package enum

import "fmt"

// ConnectorType represents the type of connector.
type ConnectorType string

const (
	// ConnectorTypeGithub is a github connector.
	ConnectorTypeGithub ConnectorType = "github"
)

func ParseConnectorType(s string) (ConnectorType, error) {
	switch s {
	case "github":
		return ConnectorTypeGithub, nil
	default:
		return "", fmt.Errorf("unknown connector type provided: %s", s)
	}
}

func (t ConnectorType) String() string {
	switch t {
	case ConnectorTypeGithub:
		return "github"
	default:
		return undefined
	}
}

func (t ConnectorType) IsSCM() bool {
	switch t {
	case ConnectorTypeGithub:
		return true
	default:
		return false
	}
}

func GetAllConnectorTypes() ([]ConnectorType, ConnectorType) {
	return connectorTypes, "" // No default value
}

var connectorTypes = sortEnum([]ConnectorType{
	ConnectorTypeGithub,
})

func (ConnectorType) Enum() []interface{}               { return toInterfaceSlice(connectorTypes) }
func (t ConnectorType) Sanitize() (ConnectorType, bool) { return Sanitize(t, GetAllConnectorTypes) }
