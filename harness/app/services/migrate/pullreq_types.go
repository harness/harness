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

package migrate

import migratetypes "github.com/harness/harness-migrate/types"

type ExternalPullRequest = migratetypes.PullRequestData
type ExternalComment = migratetypes.Comment
type ExternalReview = migratetypes.Review
type ExternalReviewer = migratetypes.Reviewer

type externalCommentThread struct {
	TopLevel ExternalComment
	Replies  []ExternalComment
}

const (
	InfoCommentMessage = "This pull request has been imported. Non-existent users who were originally listed " +
		"as the pull request author or commenter have been replaced by the principal '%s' which performed the migration.\n" +
		"Unknown emails: %v"
	MaxNumberOfUnknownEmails = 500 // limit keeping unknown users to avoid info comment text exceed ~1000 characters
)
