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

import "strings"

// WebhookAttr defines webhook attributes that can be used for sorting and filtering.
type WebhookAttr int

const (
	WebhookAttrNone WebhookAttr = iota
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	WebhookAttrID
	// TODO [CODE-1363]: remove after identifier migration.
	WebhookAttrUID
	WebhookAttrIdentifier
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	WebhookAttrDisplayName
	WebhookAttrCreated
	WebhookAttrUpdated
)

// ParseWebhookAttr parses the webhook attribute string
// and returns the equivalent enumeration.
func ParseWebhookAttr(s string) WebhookAttr {
	switch strings.ToLower(s) {
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	case id:
		return WebhookAttrID
	// TODO [CODE-1363]: remove after identifier migration.
	case uid:
		return WebhookAttrUID
	case identifier:
		return WebhookAttrIdentifier
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	case displayName:
		return WebhookAttrDisplayName
	case created, createdAt:
		return WebhookAttrCreated
	case updated, updatedAt:
		return WebhookAttrUpdated
	default:
		return WebhookAttrNone
	}
}

// String returns the string representation of the attribute.
func (a WebhookAttr) String() string {
	switch a {
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	case WebhookAttrID:
		return id
	// TODO [CODE-1363]: remove after identifier migration.
	case WebhookAttrUID:
		return uid
	case WebhookAttrIdentifier:
		return identifier
	// TODO [CODE-1364]: Remove once UID/Identifier migration is completed.
	case WebhookAttrDisplayName:
		return displayName
	case WebhookAttrCreated:
		return created
	case WebhookAttrUpdated:
		return updated
	case WebhookAttrNone:
		return ""
	default:
		return undefined
	}
}

// WebhookParent defines different types of parents of a webhook.
type WebhookParent string

func (WebhookParent) Enum() []any { return toInterfaceSlice(webhookParents) }

const (
	// WebhookParentRepo describes a repo as webhook owner.
	WebhookParentRepo WebhookParent = "repo"

	// WebhookParentSpace describes a space as webhook owner.
	WebhookParentSpace WebhookParent = "space"

	// WebhookParentRegistry describes a registry as webhook owner.
	WebhookParentRegistry WebhookParent = "registry"
)

var webhookParents = sortEnum([]WebhookParent{
	WebhookParentRepo,
	WebhookParentSpace,
	WebhookParentRegistry,
})

// WebhookExecutionResult defines the different results of a webhook execution.
type WebhookExecutionResult string

func (WebhookExecutionResult) Enum() []any { return toInterfaceSlice(webhookExecutionResults) }

const (
	// WebhookExecutionResultSuccess describes a webhook execution result that succeeded.
	WebhookExecutionResultSuccess WebhookExecutionResult = "success"

	// WebhookExecutionResultRetriableError describes a webhook execution result that failed with a retriable error.
	WebhookExecutionResultRetriableError WebhookExecutionResult = "retriable_error"

	// WebhookExecutionResultFatalError describes a webhook execution result that failed with an unrecoverable error.
	WebhookExecutionResultFatalError WebhookExecutionResult = "fatal_error"
)

var webhookExecutionResults = sortEnum([]WebhookExecutionResult{
	WebhookExecutionResultSuccess,
	WebhookExecutionResultRetriableError,
	WebhookExecutionResultFatalError,
})

// WebhookType defines different types of a webhook.
type WebhookType int

func (WebhookType) Enum() []any { return toInterfaceSlice(webhookTypes) }

const (
	// WebhookTypeExternal describes a webhook url pointing to external source.
	WebhookTypeExternal WebhookType = iota

	// WebhookTypeInternal describes a repo webhook url pointing to internal url.
	WebhookTypeInternal

	// WebhookTypeJira describes a webhook url pointing to jira.
	WebhookTypeJira
)

var webhookTypes = sortEnum([]WebhookType{
	WebhookTypeExternal,
	WebhookTypeInternal,
	WebhookTypeJira,
})

// WebhookTrigger defines the different types of webhook triggers available.
type WebhookTrigger string

func (WebhookTrigger) Enum() []any                        { return toInterfaceSlice(webhookTriggers) }
func (s WebhookTrigger) Sanitize() (WebhookTrigger, bool) { return Sanitize(s, GetAllWebhookTriggers) }

func GetAllWebhookTriggers() ([]WebhookTrigger, WebhookTrigger) {
	return webhookTriggers, "" // No default value
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

	// WebhookTriggerPullReqCreated gets triggered when a pull request gets created.
	WebhookTriggerPullReqCreated WebhookTrigger = "pullreq_created"
	// WebhookTriggerPullReqReopened gets triggered when a pull request gets reopened.
	WebhookTriggerPullReqReopened WebhookTrigger = "pullreq_reopened"
	// WebhookTriggerPullReqBranchUpdated gets triggered when a pull request source branch gets updated.
	WebhookTriggerPullReqBranchUpdated WebhookTrigger = "pullreq_branch_updated"
	// WebhookTriggerPullReqClosed gets triggered when a pull request is closed.
	WebhookTriggerPullReqClosed WebhookTrigger = "pullreq_closed"
	// WebhookTriggerPullReqCommentCreated gets triggered when a pull request comment gets created.
	WebhookTriggerPullReqCommentCreated WebhookTrigger = "pullreq_comment_created"
	// WebhookTriggerPullReqCommentUpdated gets triggered when a pull request comment gets edited.
	WebhookTriggerPullReqCommentUpdated WebhookTrigger = "pullreq_comment_updated"
	// WebhookTriggerPullReqCommentStatusUpdated gets triggered when a pull request comment status gets updated.
	WebhookTriggerPullReqCommentStatusUpdated WebhookTrigger = "pullreq_comment_status_updated"
	// WebhookTriggerPullReqMerged gets triggered when a pull request is merged.
	WebhookTriggerPullReqMerged WebhookTrigger = "pullreq_merged"
	// WebhookTriggerPullReqUpdated gets triggered when a pull request gets updated.
	WebhookTriggerPullReqUpdated WebhookTrigger = "pullreq_updated"
	// WebhookTriggerPullReqLabelAssigned gets triggered when a label is assigned to a pull request.
	WebhookTriggerPullReqLabelAssigned WebhookTrigger = "pullreq_label_assigned"
	// WebhookTriggerPullReqReviewSubmitted gets triggered when a pull request review is submitted.
	WebhookTriggerPullReqReviewSubmitted = "pullreq_review_submitted"
	// WebhookTriggerPullReqTargetBranchChanged gets triggered when a pull request target branch is changed.
	WebhookTriggerPullReqTargetBranchChanged = "pullreq_target_branch_changed"

	// WebhookTriggerArtifactCreated gets triggered when an artifact gets created.
	WebhookTriggerArtifactCreated WebhookTrigger = "artifact_created"
	// WebhookTriggerArtifactDeleted gets triggered when an artifact gets deleted.
	WebhookTriggerArtifactDeleted WebhookTrigger = "artifact_deleted"
)

var webhookTriggers = sortEnum([]WebhookTrigger{
	WebhookTriggerBranchCreated,
	WebhookTriggerBranchUpdated,
	WebhookTriggerBranchDeleted,
	WebhookTriggerTagCreated,
	WebhookTriggerTagUpdated,
	WebhookTriggerTagDeleted,
	WebhookTriggerPullReqCreated,
	WebhookTriggerPullReqUpdated,
	WebhookTriggerPullReqReopened,
	WebhookTriggerPullReqBranchUpdated,
	WebhookTriggerPullReqClosed,
	WebhookTriggerPullReqCommentCreated,
	WebhookTriggerPullReqCommentUpdated,
	WebhookTriggerPullReqCommentStatusUpdated,
	WebhookTriggerPullReqMerged,
	WebhookTriggerPullReqLabelAssigned,
	WebhookTriggerPullReqReviewSubmitted,
	WebhookTriggerPullReqTargetBranchChanged,
	WebhookTriggerArtifactCreated,
	WebhookTriggerArtifactDeleted,
})
