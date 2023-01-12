// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// WebhookParent defines different types of parents of a webhook.
type WebhookParent string

func (WebhookParent) Enum() []interface{} {
	return toInterfaceSlice(GetAllWebhookParents())
}

const (
	// WebhookParentRepo describes a repo as webhook owner.
	WebhookParentRepo WebhookParent = "repo"

	// WebhookParentSpace describes a space as webhook owner.
	WebhookParentSpace WebhookParent = "space"
)

func GetAllWebhookParents() []WebhookParent {
	return []WebhookParent{
		WebhookParentRepo,
		WebhookParentSpace,
	}
}

// WebhookExecutionResult defines the different results of a webhook execution.
type WebhookExecutionResult string

func (WebhookExecutionResult) Enum() []interface{} {
	return toInterfaceSlice(GetAllWebhookExecutionResults())
}

const (
	// WebhookExecutionResultSuccess describes a webhook execution result that succeeded.
	WebhookExecutionResultSuccess WebhookExecutionResult = "success"

	// WebhookExecutionResultRetriableError describes a webhook execution result that failed with a retriable error.
	WebhookExecutionResultRetriableError WebhookExecutionResult = "retriable_error"

	// WebhookExecutionResultFatalError describes a webhook execution result that failed with an unrecoverable error.
	WebhookExecutionResultFatalError WebhookExecutionResult = "fatal_error"
)

func GetAllWebhookExecutionResults() []WebhookExecutionResult {
	return []WebhookExecutionResult{
		WebhookExecutionResultSuccess,
		WebhookExecutionResultRetriableError,
		WebhookExecutionResultFatalError,
	}
}

// WebhookTrigger defines the different types of webhook triggers available.
type WebhookTrigger string

func (WebhookTrigger) Enum() []interface{} {
	return toInterfaceSlice(GetAllWebhookTriggers())
}

const (
	// WebhookTriggerBranchCreated gets triggered when a branch gets created.
	WebhookTriggerBranchCreated WebhookTrigger = "branch_created"
	// WebhookTriggerBranchUpdated gets triggered when a branch gets updated.
	WebhookTriggerBranchUpdated WebhookTrigger = "branch_updated"
	// WebhookTriggerBranchDeleted gets triggered when a branch gets deleted.
	WebhookTriggerBranchDeleted WebhookTrigger = "branch_deleted"

	// WebhookTriggerTagCreated gets triggered when a tag gets created.
	WebhookTriggerTagCreated WebhookTrigger = "tag_created"
	// WebhookTriggerTagUpdated gets triggered when a tag gets updated.
	WebhookTriggerTagUpdated WebhookTrigger = "tag_updated"
	// WebhookTriggerTagDeleted gets triggered when a tag gets deleted.
	WebhookTriggerTagDeleted WebhookTrigger = "tag_deleted"
)

func GetAllWebhookTriggers() []WebhookTrigger {
	return []WebhookTrigger{
		WebhookTriggerBranchCreated,
		WebhookTriggerBranchUpdated,
		WebhookTriggerBranchDeleted,
		WebhookTriggerTagCreated,
		WebhookTriggerTagUpdated,
		WebhookTriggerTagDeleted,
	}
}

var rawWebhookTriggers = toSortedStrings(GetAllWebhookTriggers())

// ParseWebhookTrigger parses the webhook trigger type.
func ParseWebhookTrigger(s string) (WebhookTrigger, bool) {
	if existsInSortedSlice(rawWebhookTriggers, s) {
		return WebhookTrigger(s), true
	}
	return "", false
}
