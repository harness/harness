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

// SSEType defines the kind of server sent event.
type SSEType string

// Enums for event types delivered to the event stream for the UI.
const (

	// Executions.

	SSETypeExecutionUpdated   SSEType = "execution_updated"
	SSETypeExecutionRunning   SSEType = "execution_running"
	SSETypeExecutionCompleted SSEType = "execution_completed"
	SSETypeExecutionCanceled  SSEType = "execution_canceled"

	// Repo import/export.

	SSETypeRepositoryImportCompleted SSEType = "repository_import_completed"
	SSETypeRepositoryExportCompleted SSEType = "repository_export_completed"

	// Pull reqs.

	SSETypePullReqUpdated SSEType = "pullreq_updated"

	SSETypePullReqReviewerAdded    SSEType = "pullreq_reviewer_added"
	SSETypePullReqtReviewerRemoved SSEType = "pullreq_reviewer_removed"

	SSETypePullReqCommentCreated SSEType = "pullreq_comment_created"
	SSETypePullReqCommentEdited  SSEType = "pullreq_comment_edited"
	SSETypePullReqCommentUpdated SSEType = "pullreq_comment_updated"

	SSETypePullReqCommentStatusResolved    SSEType = "pullreq_comment_status_resolved"
	SSETypePullReqCommentStatusReactivated SSEType = "pullreq_comment_status_reactivated"

	SSETypePullReqOpened         SSEType = "pullreq_opened"
	SSETypePullReqClosed         SSEType = "pullreq_closed"
	SSETypePullReqMarkedAsDraft  SSEType = "pullreq_marked_as_draft"
	SSETypePullReqReadyForReview SSEType = "pullreq_ready_for_review"

	// Branches.

	SSETypeBranchMergableUpdated SSEType = "branch_mergable_updated"

	SSETypeBranchCreated SSEType = "branch_created"
	SSETypeBranchUpdated SSEType = "branch_updated"
	SSETypeBranchDeleted SSEType = "branch_deleted"

	// Tags.

	SSETypeTagCreated SSEType = "tag_created"
	SSETypeTagUpdated SSEType = "tag_updated"
	SSETypeTagDeleted SSEType = "tag_deleted"

	// Statuses.

	SSETypeStatusCheckReportUpdated SSEType = "status_check_report_updated"

	// Logs.

	SSETypeLogLineAppended SSEType = "log_line_appended"

	// Rules.

	SSETypeRuleCreated SSEType = "rule_created"
	SSETypeRuleUpdated SSEType = "rule_updated"
	SSETypeRuleDeleted SSEType = "rule_deleted"

	// Webhooks.

	SSETypeWebhookCreated SSEType = "webhook_created"
	SSETypeWebhookUpdated SSEType = "webhook_updated"
	SSETypeWebhookDeleted SSEType = "webhook_deleted"
)
