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
	"github.com/harness/gitness/types/enum"
)

type Connector struct {
	ID               int64                `json:"-"`
	Description      string               `json:"description"`
	SpaceID          int64                `json:"space_id"`
	Identifier       string               `json:"identifier"`
	CreatedBy        int64                `json:"created_by"`
	Type             enum.ConnectorType   `json:"type"`
	LastTestAttempt  int64                `json:"last_test_attempt"`
	LastTestErrorMsg string               `json:"last_test_error_msg"`
	LastTestStatus   enum.ConnectorStatus `json:"last_test_status"`
	Created          int64                `json:"created"`
	Updated          int64                `json:"updated"`
	Version          int64                `json:"-"`

	// Pointers to connector specific data
	ConnectorConfig
}
