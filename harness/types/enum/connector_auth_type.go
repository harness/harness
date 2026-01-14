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

// ConnectorAuthType represents the type of connector authentication.
type ConnectorAuthType string

const (
	ConnectorAuthTypeBasic  ConnectorAuthType = "basic"
	ConnectorAuthTypeBearer ConnectorAuthType = "bearer"
)

func ParseConnectorAuthType(s string) (ConnectorAuthType, error) {
	switch s {
	case "basic":
		return ConnectorAuthTypeBasic, nil
	case "bearer":
		return ConnectorAuthTypeBearer, nil
	default:
		return "", fmt.Errorf("unknown connector auth type provided: %s", s)
	}
}

func (t ConnectorAuthType) String() string {
	switch t {
	case ConnectorAuthTypeBasic:
		return "basic"
	case ConnectorAuthTypeBearer:
		return "bearer"
	default:
		return "undefined"
	}
}

func GetAllConnectorAuthTypes() []ConnectorAuthType {
	return []ConnectorAuthType{
		ConnectorAuthTypeBasic,
		ConnectorAuthTypeBearer,
	}
}

func (ConnectorAuthType) Enum() []interface{} { return toInterfaceSlice(GetAllConnectorAuthTypes()) }
func (t ConnectorAuthType) Sanitize() (ConnectorAuthType, bool) {
	return Sanitize(t, func() ([]ConnectorAuthType, ConnectorAuthType) {
		return GetAllConnectorAuthTypes(), ""
	})
}
