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
	"github.com/harness/gitness/types/enum"
)

// InsightsProject is an error-reporting project of a repository. SDKs report
// to it using a DSN derived from its ID and public key.
type InsightsProject struct {
	ID                 int64   `json:"id"`
	RepoID             int64   `json:"repo_id"`
	Identifier         string  `json:"identifier"`
	PublicKey          string  `json:"public_key"`
	Enabled            bool    `json:"enabled"`
	RateLimitPerMinute int     `json:"rate_limit_per_minute"`
	SampleRate         float64 `json:"sample_rate"`
	CreatedBy          int64   `json:"created_by"`
	Created            int64   `json:"created"`
	Updated            int64   `json:"updated"`
}

// InsightsProjectWithDSN is an InsightsProject including the DSN to configure SDKs.
type InsightsProjectWithDSN struct {
	InsightsProject
	DSN string `json:"dsn"`
}

// InsightsProjectCreateInput is the input for creating an Insights project.
type InsightsProjectCreateInput struct {
	Identifier         string   `json:"identifier"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
	SampleRate         *float64 `json:"sample_rate,omitempty"`
}

// InsightsIssue is a group of similar events (same fingerprint).
type InsightsIssue struct {
	ID                int64                    `json:"id"`
	Version           int64                    `json:"-"`
	ProjectID         int64                    `json:"project_id"`
	RepoID            int64                    `json:"repo_id"`
	Fingerprint       string                   `json:"fingerprint"`
	Title             string                   `json:"title"`
	Culprit           string                   `json:"culprit"`
	Level             string                   `json:"level"`
	Platform          string                   `json:"platform"`
	FirstSeen         int64                    `json:"first_seen"`
	LastSeen          int64                    `json:"last_seen"`
	TimesSeen         int64                    `json:"times_seen"`
	UsersSeen         int64                    `json:"users_seen"`
	LastUser          string                   `json:"-"`
	FirstRelease      string                   `json:"first_release"`
	LastRelease       string                   `json:"last_release"`
	Status            enum.InsightsIssueStatus `json:"status"`
	ResolvedInRelease string                   `json:"resolved_in_release,omitempty"`
	TaskID            *int64                   `json:"task_id,omitempty"`
	Created           int64                    `json:"created"`
	Updated           int64                    `json:"updated"`
}

// InsightsIssueFilter stores Insights issue list query parameters.
type InsightsIssueFilter struct {
	ListQueryFilter
	Statuses []enum.InsightsIssueStatus `json:"status"`
}

// InsightsEvent is a single stored occurrence of an issue.
type InsightsEvent struct {
	ID          int64  `json:"id"`
	IssueID     int64  `json:"issue_id"`
	ProjectID   int64  `json:"project_id"`
	EventID     string `json:"event_id"`
	Timestamp   int64  `json:"timestamp"`
	Release     string `json:"release"`
	Environment string `json:"environment"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	// Payload is the sanitized event as JSON (size-capped).
	Payload string `json:"payload,omitempty"`
	Created int64  `json:"created"`
}
