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
	ID         int64              `json:"id" yaml:"id"`
	Version    int64              `json:"version" yaml:"-"`
	ParentID   int64              `json:"parent_id" yaml:"-"`
	ParentType enum.WebhookParent `json:"parent_type" yaml:"-"`
	CreatedBy  int64              `json:"created_by" yaml:"created_by"`
	Created    int64              `json:"created" yaml:"created"`
	Updated    int64              `json:"updated" yaml:"updated"`
	Type       enum.WebhookType   `json:"-"`

	// scope 0 indicates repo; scope > 0 indicates space depth level
	Scope int64 `json:"scope"`

	Identifier string `json:"identifier"`
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	DisplayName           string                       `json:"display_name" yaml:"display_name"`
	Description           string                       `json:"description" yaml:"description"`
	URL                   string                       `json:"url" yaml:"url"`
	Secret                string                       `json:"-" yaml:"-"`
	Enabled               bool                         `json:"enabled" yaml:"enabled"`
	Insecure              bool                         `json:"insecure" yaml:"insecure"`
	Triggers              []enum.WebhookTrigger        `json:"triggers" yaml:"triggers"`
	LatestExecutionResult *enum.WebhookExecutionResult `json:"latest_execution_result,omitempty" yaml:"-"`
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

// Clone makes a deep copy of the webhook object.
func (w Webhook) Clone() Webhook {
	webhook := w

	// Deep copy the LatestExecutionResult pointer if it exists
	if w.LatestExecutionResult != nil {
		result := *w.LatestExecutionResult
		webhook.LatestExecutionResult = &result
	}

	// Deep copy the Triggers slice
	if len(w.Triggers) > 0 {
		triggers := make([]enum.WebhookTrigger, len(w.Triggers))
		copy(triggers, w.Triggers)
		webhook.Triggers = triggers
	}

	return webhook
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

type WebhookSignatureMetadata struct {
	Signature string
	BodyBytes []byte
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

// WebhookCore represents a webhook DTO object.
type WebhookCore struct {
	ID                    int64
	Version               int64
	ParentID              int64
	ParentType            enum.WebhookParent
	CreatedBy             int64
	Created               int64
	Updated               int64
	Type                  enum.WebhookType
	Scope                 int64
	Identifier            string
	DisplayName           string
	Description           string
	URL                   string
	Secret                string
	Enabled               bool
	Insecure              bool
	Triggers              []enum.WebhookTrigger
	LatestExecutionResult *enum.WebhookExecutionResult
	SecretIdentifier      string
	SecretSpaceID         int64
	ExtraHeaders          []ExtraHeader
}

// WebhookExecutionCore represents a webhook execution DTO object.
type WebhookExecutionCore struct {
	ID            int64
	RetriggerOf   *int64
	Retriggerable bool
	Created       int64
	WebhookID     int64
	TriggerType   enum.WebhookTrigger
	TriggerID     string
	Result        enum.WebhookExecutionResult
	Duration      int64
	Error         string
	Request       WebhookExecutionRequest
	Response      WebhookExecutionResponse
}

type ExtraHeader struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
