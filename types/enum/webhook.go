// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "sort"

// WebhookParent defines different types of parents of a webhook.
type WebhookParent string

const (
	// WebhookParentSpace describes a space as webhook owner.
	WebhookParentSpace WebhookParent = "space"

	// WebhookParentSpace describes a repo as webhook owner.
	WebhookParentRepo WebhookParent = "repo"
)

// WebhookExecutionResult defines the different results of a webhook execution.
type WebhookExecutionResult string

const (
	// WebhookExecutionResultFatalError describes a webhook execution result that failed with an unrecoverable error.
	WebhookExecutionResultFatalError WebhookExecutionResult = "fatal_error"

	// WebhookExecutionResultRetriableError describes a webhook execution result that failed with a retriable error.
	WebhookExecutionResultRetriableError WebhookExecutionResult = "retriable_error"

	// WebhookExecutionResultSuccess describes a webhook execution result that succeeded.
	WebhookExecutionResultSuccess WebhookExecutionResult = "success"
)

// WebhookTrigger defines the different types of webhook triggers available.
// NOTE: For now we keep a small list - will be extended later on once we decided on a final set of triggers.
type WebhookTrigger string

const (
	// WebhookTriggerPush describes a push trigger.
	WebhookTriggerPush WebhookTrigger = "push"
)

var webhookTriggers = []string{
	string(WebhookTriggerPush),
}

func init() {
	sort.Strings(webhookTriggers)
}

// ParsePullReqActivityType parses the webhook trigger type.
func ParseWebhookTrigger(s string) (WebhookTrigger, bool) {
	if existsInSortedSlice(webhookTriggers, s) {
		return WebhookTrigger(s), true
	}
	return "", false
}
