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
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

// Webhook represents a webhook.
type Webhook struct {
	// TODO [CODE-1364]: Hide once UID/Identifier migration is completed.
	ID         int64              `json:"id"`
	Version    int64              `json:"version"`
	ParentID   int64              `json:"parent_id"`
	ParentType enum.WebhookParent `json:"parent_type"`
	CreatedBy  int64              `json:"created_by"`
	Created    int64              `json:"created"`
	Updated    int64              `json:"updated"`
	Internal   bool               `json:"-"`

	// scope 0 indicates repo; scope > 0 indicates space depth level
	Scope int64 `json:"scope"`

	Identifier string `json:"identifier"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName           string                       `json:"display_name"`
	Description           string                       `json:"description"`
	URL                   string                       `json:"url"`
	Secret                string                       `json:"-"`
	Enabled               bool                         `json:"enabled"`
	Insecure              bool                         `json:"insecure"`
	Triggers              []enum.WebhookTrigger        `json:"triggers"`
	LatestExecutionResult *enum.WebhookExecutionResult `json:"latest_execution_result,omitempty"`
}

// MarshalJSON overrides the default json marshaling for `Webhook` allowing us to inject the `HasSecret` field.
// NOTE: This is required as we don't expose the `Secret` field and thus the caller wouldn't know whether
// the webhook contains a secret or not.
// NOTE: This is used as an alternative to adding an `HasSecret` field to Webhook itself, which would
// require us to keep `HasSecret` in sync with the `Secret` field, while `HasSecret` is not used internally at all.
func (w *Webhook) MarshalJSON() ([]byte, error) {
	// WebhookAlias allows us to embed the original Webhook object (avoiding redefining all fields)
	// while avoiding an infinite loop of marsheling.
	type WebhookAlias Webhook
	return json.Marshal(&struct {
		*WebhookAlias
		HasSecret bool `json:"has_secret"`
		// TODO [CODE-1363]: remove after identifier migration.
		UID string `json:"uid"`
	}{
		WebhookAlias: (*WebhookAlias)(w),
		HasSecret:    w != nil && w.Secret != "",
		// TODO [CODE-1363]: remove after identifier migration.
		UID: w.Identifier,
	})
}

type WebhookCreateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string `json:"uid" deprecated:"true"`
	Identifier string `json:"identifier"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName string                `json:"display_name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	Secret      string                `json:"secret"`
	Enabled     bool                  `json:"enabled"`
	Insecure    bool                  `json:"insecure"`
	Triggers    []enum.WebhookTrigger `json:"triggers"`
}

type WebhookUpdateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName *string               `json:"display_name"`
	Description *string               `json:"description"`
	URL         *string               `json:"url"`
	Secret      *string               `json:"secret"`
	Enabled     *bool                 `json:"enabled"`
	Insecure    *bool                 `json:"insecure"`
	Triggers    []enum.WebhookTrigger `json:"triggers"`
}

// WebhookExecution represents a single execution of a webhook.
type WebhookExecution struct {
	ID            int64                       `json:"id"`
	RetriggerOf   *int64                      `json:"retrigger_of,omitempty"`
	Retriggerable bool                        `json:"retriggerable"`
	Created       int64                       `json:"created"`
	WebhookID     int64                       `json:"webhook_id"`
	TriggerType   enum.WebhookTrigger         `json:"trigger_type"`
	TriggerID     string                      `json:"-"`
	Result        enum.WebhookExecutionResult `json:"result"`
	Duration      int64                       `json:"duration"`
	Error         string                      `json:"error,omitempty"`
	Request       WebhookExecutionRequest     `json:"request"`
	Response      WebhookExecutionResponse    `json:"response"`
}

// WebhookExecutionRequest represents the request of a webhook execution.
type WebhookExecutionRequest struct {
	URL     string `json:"url"`
	Headers string `json:"headers"`
	Body    string `json:"body"`
}

// WebhookExecutionResponse represents the response of a webhook execution.
type WebhookExecutionResponse struct {
	StatusCode int    `json:"status_code"`
	Status     string `json:"status"`
	Headers    string `json:"headers"`
	Body       string `json:"body"`
}

// WebhookFilter stores Webhook query parameters for listing.
type WebhookFilter struct {
	Query        string           `json:"query"`
	Page         int              `json:"page"`
	Size         int              `json:"size"`
	Sort         enum.WebhookAttr `json:"sort"`
	Order        enum.Order       `json:"order"`
	SkipInternal bool             `json:"-"`
}

// WebhookExecutionFilter stores WebhookExecution query parameters for listing.
type WebhookExecutionFilter struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

type WebhookParentInfo struct {
	Type enum.WebhookParent
	ID   int64
}
