//  Copyright 2023 Harness, Inc.
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
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types/enum"
)

// Webhook DTO object.
type Webhook struct {
	ID                    int64
	Version               int64
	ParentType            enum.RegistryWebhookParent
	ParentID              int64
	CreatedBy             int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Scope                 int64
	Identifier            string
	Name                  string
	Description           string
	URL                   string
	SecretIdentifier      string
	SecretSpaceID         int
	Enabled               bool
	Insecure              bool
	Internal              bool
	ExtraHeaders          []artifact.ExtraHeader
	Triggers              []artifact.Trigger
	LatestExecutionResult *artifact.WebhookExecResult
}
