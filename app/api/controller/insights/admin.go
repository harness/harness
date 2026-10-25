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
	"strconv"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// auditProject is the audited view of a project. It never contains the public key.
type auditProject struct {
	ID                 int64   `json:"id"`
	Identifier         string  `json:"identifier"`
	Enabled            bool    `json:"enabled"`
	RateLimitPerMinute int     `json:"rate_limit_per_minute"`
	SampleRate         float64 `json:"sample_rate"`
}

func toAuditProject(p *types.InsightsProject) *auditProject {
	if p == nil {
		return nil
	}
	return &auditProject{
		ID:                 p.ID,
		Identifier:         p.Identifier,
		Enabled:            p.Enabled,
		RateLimitPerMinute: p.RateLimitPerMinute,
		SampleRate:         p.SampleRate,
	}
}

// logProject writes an audit log entry for a project change. Failures are logged only.
func (c *Controller) logProject(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	action audit.Action,
	project, oldState, newState *types.InsightsProject,
	extra ...string,
) {
	opts := []audit.Option{
		audit.WithData(append([]string{"repo_path", repo.Path}, extra...)...),
	}
	if oldState != nil {
		opts = append(opts, audit.WithOldObject(toAuditProject(oldState)))
	}
	if newState != nil {
		opts = append(opts, audit.WithNewObject(toAuditProject(newState)))
	}

	err := c.audit.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeInsightsProject, project.Identifier,
			"project_id", strconv.FormatInt(project.ID, 10)),
		action,
		paths.Parent(repo.Path),
		opts...,
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to insert audit log for insights project %s", action)
	}
}

// UpdateProject updates the settings of an Insights project.
func (c *Controller) UpdateProject(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	projectID int64,
	in *types.InsightsProjectUpdateInput,
) (*types.InsightsProjectWithDSN, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	oldState, p, err := c.service.UpdateProject(ctx, r.ID, projectID, in)
	if err != nil {
		return nil, err
	}

	c.logProject(ctx, session, r, audit.ActionUpdated, p, oldState, p)

	return c.withDSN(ctx, p), nil
}

// RotateProjectKey generates a new public key for the project and returns the project with the new DSN.
func (c *Controller) RotateProjectKey(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	projectID int64,
) (*types.InsightsProjectWithDSN, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	p, err := c.service.RotateProjectKey(ctx, r.ID, projectID)
	if err != nil {
		return nil, err
	}

	c.logProject(ctx, session, r, audit.ActionUpdated, p, nil, nil, "operation", "rotate_key")

	return c.withDSN(ctx, p), nil
}

// DeleteProject deletes the project including its issues and events.
func (c *Controller) DeleteProject(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	projectID int64,
) error {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return err
	}

	p, err := c.service.DeleteProject(ctx, r.ID, projectID)
	if err != nil {
		return err
	}

	c.logProject(ctx, session, r, audit.ActionDeleted, p, p, nil)

	return nil
}

// ListIssues lists the issues of the repository and returns the total count.
func (c *Controller) ListIssues(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.InsightsIssueFilter,
) ([]*types.InsightsIssue, int64, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, err
	}

	return c.service.ListIssues(ctx, r.ID, filter)
}

// FindIssue returns the issue.
func (c *Controller) FindIssue(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	issueID int64,
) (*types.InsightsIssue, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	return c.service.FindIssue(ctx, r.ID, issueID)
}

// ListIssueEvents lists the events of the issue (newest first) and returns the total count.
func (c *Controller) ListIssueEvents(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	issueID int64,
	pagination types.Pagination,
) ([]*types.InsightsEvent, int64, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, err
	}

	return c.service.ListIssueEvents(ctx, r.ID, issueID, pagination)
}

// UpdateIssue changes the status of the issue.
func (c *Controller) UpdateIssue(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	issueID int64,
	in *types.InsightsIssueUpdateInput,
) (*types.InsightsIssue, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	return c.service.UpdateIssue(ctx, r.ID, issueID, in)
}
