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

package insights

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const maxResolvedReleaseLength = 200

// FindProject returns the project of the repository, store.ErrResourceNotFound
// if it does not exist or belongs to another repository.
func (s *Service) FindProject(ctx context.Context, repoID, projectID int64) (*types.InsightsProject, error) {
	p, err := s.projects.Find(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find insights project: %w", err)
	}
	if p.RepoID != repoID {
		return nil, gitness_store.ErrResourceNotFound
	}
	return p, nil
}

// UpdateProject updates the mutable settings of the project and returns the old and new state.
func (s *Service) UpdateProject(
	ctx context.Context,
	repoID, projectID int64,
	in *types.InsightsProjectUpdateInput,
) (*types.InsightsProject, *types.InsightsProject, error) {
	p, err := s.FindProject(ctx, repoID, projectID)
	if err != nil {
		return nil, nil, err
	}
	old := *p

	if in.RateLimitPerMinute != nil {
		if *in.RateLimitPerMinute < 0 || *in.RateLimitPerMinute > MaxRateLimitPerMinute {
			return nil, nil, usererror.BadRequestf("rate_limit_per_minute must be between 0 and %d.",
				MaxRateLimitPerMinute)
		}
		p.RateLimitPerMinute = *in.RateLimitPerMinute
	}
	if in.SampleRate != nil {
		if *in.SampleRate < 0 || *in.SampleRate > 1 {
			return nil, nil, usererror.BadRequest("sample_rate must be between 0 and 1.")
		}
		p.SampleRate = *in.SampleRate
	}
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}
	p.Updated = s.now().UnixMilli()

	if err = s.projects.Update(ctx, p); err != nil {
		return nil, nil, fmt.Errorf("failed to update insights project: %w", err)
	}
	return &old, p, nil
}

// RotateProjectKey replaces the public key of the project. The previous key stops working immediately.
func (s *Service) RotateProjectKey(ctx context.Context, repoID, projectID int64) (*types.InsightsProject, error) {
	p, err := s.FindProject(ctx, repoID, projectID)
	if err != nil {
		return nil, err
	}

	key, err := newPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}
	p.PublicKey = key
	p.Updated = s.now().UnixMilli()

	if err = s.projects.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to update insights project: %w", err)
	}
	return p, nil
}

// DeleteProject deletes the project including its issues and events and returns it.
func (s *Service) DeleteProject(ctx context.Context, repoID, projectID int64) (*types.InsightsProject, error) {
	p, err := s.FindProject(ctx, repoID, projectID)
	if err != nil {
		return nil, err
	}
	if err = s.projects.Delete(ctx, p.ID); err != nil {
		return nil, fmt.Errorf("failed to delete insights project: %w", err)
	}
	return p, nil
}

// ListIssues returns a page of the issues of the repository and the total count.
func (s *Service) ListIssues(
	ctx context.Context,
	repoID int64,
	filter *types.InsightsIssueFilter,
) ([]*types.InsightsIssue, int64, error) {
	list, err := s.issues.List(ctx, repoID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list insights issues: %w", err)
	}
	count, err := s.issues.Count(ctx, repoID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count insights issues: %w", err)
	}
	return list, count, nil
}

// FindIssue returns the issue of the repository.
func (s *Service) FindIssue(ctx context.Context, repoID, issueID int64) (*types.InsightsIssue, error) {
	i, err := s.issues.Find(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to find insights issue: %w", err)
	}
	if i.RepoID != repoID {
		return nil, gitness_store.ErrResourceNotFound
	}
	return i, nil
}

// ListIssueEvents returns a page of the events of the issue (newest first) and the total count.
func (s *Service) ListIssueEvents(
	ctx context.Context,
	repoID, issueID int64,
	pagination types.Pagination,
) ([]*types.InsightsEvent, int64, error) {
	if _, err := s.FindIssue(ctx, repoID, issueID); err != nil {
		return nil, 0, err
	}
	list, err := s.events.List(ctx, issueID, pagination)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list insights events: %w", err)
	}
	count, err := s.events.Count(ctx, issueID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count insights events: %w", err)
	}
	return list, count, nil
}

// UpdateIssue sets the status of the issue to resolved, ignored or unresolved.
func (s *Service) UpdateIssue(
	ctx context.Context,
	repoID, issueID int64,
	in *types.InsightsIssueUpdateInput,
) (*types.InsightsIssue, error) {
	switch in.Status {
	case enum.InsightsIssueStatusResolved, enum.InsightsIssueStatusIgnored, enum.InsightsIssueStatusUnresolved:
	default:
		return nil, usererror.BadRequest("status must be one of resolved, ignored and unresolved.")
	}
	if in.ResolvedInRelease != nil {
		if *in.ResolvedInRelease != "" && in.Status != enum.InsightsIssueStatusResolved {
			return nil, usererror.BadRequest("resolved_in_release can only be set when status is resolved.")
		}
		if len(*in.ResolvedInRelease) > maxResolvedReleaseLength {
			return nil, usererror.BadRequest("resolved_in_release is too long.")
		}
	}

	for range maxUpsertRetries {
		issue, err := s.FindIssue(ctx, repoID, issueID)
		if err != nil {
			return nil, err
		}

		issue.Status = in.Status
		issue.ResolvedInRelease = ""
		if in.Status == enum.InsightsIssueStatusResolved && in.ResolvedInRelease != nil {
			issue.ResolvedInRelease = *in.ResolvedInRelease
		}
		issue.Updated = s.now().UnixMilli()

		if err = s.issues.Update(ctx, issue); err != nil {
			if errors.Is(err, gitness_store.ErrVersionConflict) {
				continue
			}
			return nil, fmt.Errorf("failed to update insights issue: %w", err)
		}
		return issue, nil
	}

	return nil, errors.New("insights: too many concurrent updates of the issue")
}
