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

// ConnectorStatus represents the status of the connector after testing connection.
type ConnectorStatus string

const (
	ConnectorStatusSuccess ConnectorStatus = "success"

	ConnectorStatusFailed ConnectorStatus = "failed"
)

func (s ConnectorStatus) String() string {
	switch s {
	case ConnectorStatusSuccess:
		return "success"
	case ConnectorStatusFailed:
		return "failed"
	default:
		return undefined
	}
}

func GetAllConnectorStatus() ([]ConnectorStatus, ConnectorStatus) {
	return connectorStatus, "" // No default value
}

var connectorStatus = sortEnum([]ConnectorStatus{
	ConnectorStatusSuccess,
	ConnectorStatusFailed,
})

func (ConnectorStatus) Enum() []interface{} { return toInterfaceSlice(connectorStatus) }
func (s ConnectorStatus) Sanitize() (ConnectorStatus, bool) {
	return Sanitize(s, GetAllConnectorStatus)
}
