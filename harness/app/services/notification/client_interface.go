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

package notification

import (
	"context"

	"github.com/harness/gitness/types"
)

// Client is an interface for sending notifications, such as emails, Slack messages etc.
// It is implemented by MailClient and in future we can have other implementations for other channels like Slack etc.
type Client interface {
	SendCommentPRAuthor(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *CommentPayload,
	) error
	SendCommentMentions(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *CommentPayload,
	) error
	SendCommentParticipants(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *CommentPayload,
	) error
	SendReviewerAdded(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *ReviewerAddedPayload,
	) error
	SendPullReqBranchUpdated(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *PullReqBranchUpdatedPayload,
	) error
	SendReviewSubmitted(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *ReviewSubmittedPayload,
	) error
	SendPullReqStateChanged(
		ctx context.Context,
		recipients []*types.PrincipalInfo,
		payload *PullReqStateChangedPayload,
	) error
}
