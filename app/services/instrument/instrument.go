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

package instrument

import (
	"context"
	"time"

	"github.com/harness/gitness/types"
)

type CreationType string

const (
	CreationTypeCreate CreationType = "CREATE"
	CreationTypeImport CreationType = "IMPORT"
)

type Property string

const (
	PropertyRepositoryID              Property = "repository_id"
	PropertyRepositoryName            Property = "repository_name"
	PropertyRepositoryCreationType    Property = "creation_type"
	PropertyPullRequestID             Property = "pull_request_id"
	PropertyCommentIsReplied          Property = "comment_is_replied"
	PropertyCommentContainsSuggestion Property = "contains_suggestion"
	PropertyMergeStrategy             Property = "merge_strategy"
	PropertyRuleID                    Property = "rule_id"
	PropertyIsDefaultBranch           Property = "is_default_branch"
	PropertyDecision                  Property = "decision"
	PropertyRepositories              Property = "repositories"

	PropertySpaceID   Property = "space_id"
	PropertySpaceName Property = "space_name"
)

type EventType string

const (
	EventTypeRepositoryCreate    EventType = "Repository create"
	EventTypeRepositoryCount     EventType = "Repository count"
	EventTypeCommitCount         EventType = "Commit count"
	EventTypeCreateCommit        EventType = "Create commit"
	EventTypeCreateBranch        EventType = "Create branch"
	EventTypeCreateTag           EventType = "Create tag"
	EventTypeCreatePullRequest   EventType = "Create pull request"
	EventTypeMergePullRequest    EventType = "Merge pull request"
	EventTypeReviewPullRequest   EventType = "Review pull request"
	EventTypeCreatePRComment     EventType = "Create PR comment"
	EventTypeCreateBranchRule    EventType = "Create branch rule"
	EventTypePRSuggestionApplied EventType = "Pull request suggestion applied"
)

func (e EventType) String() string {
	return string(e)
}

type Event struct {
	Type       EventType            `json:"event"`
	Category   string               `json:"category"`
	Principal  *types.PrincipalInfo `json:"user_id,omitempty"`
	GroupID    string               `json:"group_id,omitempty"`
	Timestamp  time.Time            `json:"timestamp,omitempty"`
	Path       string               `json:"path"`
	RemoteAddr string               `json:"remote_addr"`
	Properties map[Property]any     `json:"properties,omitempty"`
}

type Service interface {
	Track(ctx context.Context, event Event) error
	Close(ctx context.Context) error
}
