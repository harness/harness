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

// Package shield provides the controller of the repository scoped Shield API.
package shield

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/shield"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

const maxReasonLength = 1024

// Scanner is the part of the Shield service used by the controller.
type Scanner interface {
	Enabled() bool
	Enqueue(
		ctx context.Context,
		repo *types.RepositoryCore,
		commitSHA string,
		trigger enum.ShieldScanTrigger,
	) (*types.ShieldScan, error)
}

var _ Scanner = (*shield.Service)(nil)

// Controller handles the Shield API.
type Controller struct {
	authorizer authz.Authorizer
	repoFinder refcache.RepoFinder
	git        git.Interface
	service    Scanner
	scans      store.ShieldScanStore
	findings   store.ShieldFindingStore
}

// NewController returns a new Controller.
func NewController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	gitSvc git.Interface,
	service Scanner,
	scans store.ShieldScanStore,
	findings store.ShieldFindingStore,
) *Controller {
	return &Controller{
		authorizer: authorizer,
		repoFinder: repoFinder,
		git:        gitSvc,
		service:    service,
		scans:      scans,
		findings:   findings,
	}
}

// ProvideController provides the Shield controller.
func ProvideController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	gitSvc git.Interface,
	service *shield.Service,
	scans store.ShieldScanStore,
	findings store.ShieldFindingStore,
) *Controller {
	return NewController(authorizer, repoFinder, gitSvc, service, scans, findings)
}

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.RepositoryCore, error) {
	r, err := repo.GetRepo(ctx, c.repoFinder, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, r, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return r, nil
}

// ScanInput is the input of CreateScan.
type ScanInput struct {
	// CommitSHA is the commit to scan; the head of the default branch if empty.
	CommitSHA string `json:"commit_sha"`
}

// CreateScan schedules a manual scan of the repository.
func (c *Controller) CreateScan(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *ScanInput,
) (*types.ShieldScan, error) {
	if !c.service.Enabled() {
		return nil, usererror.BadRequest("Shield is not enabled")
	}
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	sha := ""
	if in != nil {
		sha = strings.TrimSpace(in.CommitSHA)
	}
	if sha == "" {
		out, gerr := c.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: git.ReadParams{RepoUID: r.GitUID},
			BranchName: r.DefaultBranch,
		})
		if gerr != nil {
			return nil, fmt.Errorf("failed to get head of default branch: %w", gerr)
		}
		sha = out.Branch.SHA.String()
	}

	return c.service.Enqueue(ctx, r, sha, enum.ShieldScanTriggerManual)
}

// ListScans lists the scans of the repository, newest first.
func (c *Controller) ListScans(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	page, size int,
) ([]*types.ShieldScan, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}
	return c.scans.ListByRepo(ctx, r.ID, page, size)
}

// ListFindings lists the findings of the repository and their total count.
func (c *Controller) ListFindings(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.ShieldFindingFilter,
) ([]*types.ShieldFinding, int64, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, err
	}
	list, err := c.findings.List(ctx, r.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shield findings: %w", err)
	}
	total, err := c.findings.Count(ctx, r.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count shield findings: %w", err)
	}
	return list, total, nil
}

// FindingUpdateInput is the input of UpdateFinding.
type FindingUpdateInput struct {
	State  enum.ShieldFindingState `json:"state"`
	Reason string                  `json:"reason"`
}

// UpdateFinding sets a finding to ignored or accepted_risk, with a reason.
func (c *Controller) UpdateFinding(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	findingID int64,
	in *FindingUpdateInput,
) (*types.ShieldFinding, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, usererror.BadRequest("A request body is required")
	}
	if in.State != enum.ShieldFindingStateIgnored && in.State != enum.ShieldFindingStateAcceptedRisk {
		return nil, usererror.BadRequestf("State must be %q or %q",
			enum.ShieldFindingStateIgnored, enum.ShieldFindingStateAcceptedRisk)
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, usererror.BadRequest("A reason is required")
	}
	if len(reason) > maxReasonLength {
		return nil, usererror.BadRequestf("Reason can't be longer than %d characters", maxReasonLength)
	}

	f, err := c.findings.Find(ctx, findingID)
	if err != nil {
		return nil, fmt.Errorf("failed to find shield finding: %w", err)
	}
	if f.RepoID != r.ID {
		return nil, usererror.ErrNotFound
	}

	if err = c.findings.UpdateState(ctx, f.ID, in.State, reason, time.Now().UnixMilli()); err != nil {
		return nil, fmt.Errorf("failed to update shield finding: %w", err)
	}
	return c.findings.Find(ctx, f.ID)
}
