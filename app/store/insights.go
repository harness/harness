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

package store

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	// InsightsProjectStore defines the Insights project data storage.
	InsightsProjectStore interface {
		// Create inserts the project and sets its ID. Identifier and public key are unique.
		Create(ctx context.Context, project *types.InsightsProject) error
		// Find returns the project with the given ID.
		Find(ctx context.Context, id int64) (*types.InsightsProject, error)
		// FindByIdentifier returns the project of the repository with the given identifier.
		FindByIdentifier(ctx context.Context, repoID int64, identifier string) (*types.InsightsProject, error)
		// List returns all projects of the repository ordered by identifier.
		List(ctx context.Context, repoID int64) ([]*types.InsightsProject, error)
		// Update updates the mutable fields (enabled, rate limit, sample rate, public key).
		Update(ctx context.Context, project *types.InsightsProject) error
		// Delete removes the project including its issues and events.
		Delete(ctx context.Context, id int64) error
	}

	// InsightsIssueStore defines the Insights issue data storage.
	InsightsIssueStore interface {
		// Create inserts the issue and sets its ID. Returns store.ErrDuplicate
		// if the fingerprint exists already in the project.
		Create(ctx context.Context, issue *types.InsightsIssue) error
		Find(ctx context.Context, id int64) (*types.InsightsIssue, error)
		FindByFingerprint(ctx context.Context, projectID int64, fingerprint string) (*types.InsightsIssue, error)
		// Update updates the issue using optimistic locking (store.ErrVersionConflict).
		Update(ctx context.Context, issue *types.InsightsIssue) error
		// List returns the issues of a repository, most recently seen first.
		List(ctx context.Context, repoID int64, filter *types.InsightsIssueFilter) ([]*types.InsightsIssue, error)
		Count(ctx context.Context, repoID int64, filter *types.InsightsIssueFilter) (int64, error)
	}

	// InsightsEventStore defines the Insights event data storage.
	InsightsEventStore interface {
		// Create inserts the event and sets its ID. Returns store.ErrDuplicate
		// if the event id exists already in the project.
		Create(ctx context.Context, event *types.InsightsEvent) error
		// Exists reports whether the event id was stored already for the project.
		Exists(ctx context.Context, projectID int64, eventID string) (bool, error)
		// ListByIssue returns the latest events of the issue, newest first.
		ListByIssue(ctx context.Context, issueID int64, limit int) ([]*types.InsightsEvent, error)
		// List returns a page of the events of the issue, newest first.
		List(ctx context.Context, issueID int64, pagination types.Pagination) ([]*types.InsightsEvent, error)
		// Count returns the number of events of the issue.
		Count(ctx context.Context, issueID int64) (int64, error)
	}
)
