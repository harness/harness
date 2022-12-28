// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"github.com/harness/gitness/types/enum"
)

// Webhook represents a webhook.
type Webhook struct {
	ID         int64              `json:"id"`
	Version    int64              `json:"version"`
	ParentID   int64              `json:"parent_id"`
	ParentType enum.WebhookParent `json:"parent_type"`
	CreatedBy  int64              `json:"created_by"`
	Created    int64              `json:"created"`
	Updated    int64              `json:"updated"`

	URL      string                `json:"url"`
	Secret   string                `json:"-"`
	Enabled  bool                  `json:"enabled"`
	Insecure bool                  `json:"insecure"`
	Triggers []enum.WebhookTrigger `json:"triggers"`
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
	StatusCode int64  `json:"status"`
	Status     string `json:"status_code"`
	Headers    string `json:"headers"`
	Body       string `json:"body"`
}

// WebhookFilter stores Webhook query parameters for listing.
type WebhookFilter struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

// WebhookExecutionFilter stores WebhookExecution query parameters for listing.
type WebhookExecutionFilter struct {
	Page int `json:"page"`
	Size int `json:"size"`
}
