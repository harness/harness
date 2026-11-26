// Copyright 2026 Harness, Inc.
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

package reposettings

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/require"
)

const cxrRepoRef = "root/repo"

type cxrAuditLog struct {
	actions []audit.Action
	err     error
}

func (a *cxrAuditLog) Log(
	_ context.Context, _ types.Principal, _ audit.Resource, action audit.Action, _ string, _ ...audit.Option,
) error {
	a.actions = append(a.actions, action)
	return a.err
}

type cxrEnv struct {
	*cxrBase
	audit *cxrAuditLog
	repo  *types.Repository
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t, "root")
	return &cxrEnv{cxrBase: b, audit: &cxrAuditLog{}, repo: b.createRepo(ctx, t, space, "repo")}
}

func (e *cxrEnv) controller(mode cxrAuthMode) *Controller {
	svc := settings.NewService(database.NewSettingsStore(e.db), e.spaceFinder)
	return NewController(cxrAuthorizer{mode: mode}, e.repoFinder, e.spaceFinder, svc, e.audit)
}

func (e *cxrEnv) createChildSpace(
	ctx context.Context, t *testing.T, parent *types.Space, identifier string,
) *types.Space {
	t.Helper()
	space := &types.Space{
		Identifier: identifier, ParentID: parent.ID, CreatedBy: e.admin.ID,
		RootSpaceID: parent.RootSpaceID, RootSpaceIdentifier: parent.RootSpaceIdentifier,
	}
	require.NoError(t, e.spaces.Create(ctx, space))
	require.NoError(t, e.spacePaths.InsertSegment(ctx, &types.SpacePathSegment{
		Identifier: identifier, SpaceID: space.ID, ParentID: parent.ID, IsPrimary: true, CreatedBy: e.admin.ID,
	}))
	return space
}

func TestCxrRepoSettingsGeneral(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	defaults, err := c.GeneralFind(ctx, e.session(), cxrRepoRef)
	require.NoError(t, err)
	require.Equal(t, settings.GetDefaultGeneralSettings(), defaults)

	updated, err := c.GeneralUpdate(ctx, e.session(), cxrRepoRef, &settings.GeneralSettings{
		FileSizeLimit: ptr.Int64(1234), GitLFSEnabled: ptr.Bool(true),
	})
	require.NoError(t, err)
	require.EqualValues(t, 1234, *updated.FileSizeLimit)
	require.True(t, *updated.GitLFSEnabled)
	require.Equal(t, []audit.Action{audit.ActionUpdated}, e.audit.actions)

	found, err := c.GeneralFind(ctx, e.session(), cxrRepoRef)
	require.NoError(t, err)
	require.Equal(t, updated, found)

	// A failing audit log does not fail the update.
	e.audit.err = errCxrAuthz
	_, err = c.GeneralUpdate(ctx, e.session(), cxrRepoRef, &settings.GeneralSettings{AutoMergeEnabled: ptr.Bool(true)})
	require.NoError(t, err)
}

func TestCxrRepoSettingsSecurity(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	c := e.controller(cxrAllow)

	defaults, err := c.SecurityFind(ctx, e.session(), cxrRepoRef)
	require.NoError(t, err)
	require.Equal(t, settings.GetDefaultSecuritySettings(), defaults)

	updated, err := c.SecurityUpdate(ctx, e.session(), cxrRepoRef, &settings.SecuritySettings{
		SecretScanningEnabled: ptr.Bool(true), PrincipalCommitterMatch: ptr.Bool(true),
	})
	require.NoError(t, err)
	require.True(t, *updated.SecretScanningEnabled)
	require.True(t, *updated.PrincipalCommitterMatch)
	require.Equal(t, []audit.Action{audit.ActionUpdated}, e.audit.actions)

	found, err := c.SecurityFind(ctx, e.session(), cxrRepoRef)
	require.NoError(t, err)
	require.Equal(t, updated, found)

	e.audit.err = errCxrAuthz
	_, err = c.SecurityUpdate(ctx, e.session(), cxrRepoRef, &settings.SecuritySettings{})
	require.NoError(t, err)
}

func TestCxrRepoSettingsAccessErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)

	_, err := allow.GeneralFind(ctx, e.session(), "root/nope")
	require.Error(t, err)
	_, err = deny.GeneralFind(ctx, e.session(), cxrRepoRef)
	require.ErrorContains(t, err, "access check failed")
	_, err = deny.GeneralUpdate(ctx, e.session(), cxrRepoRef, &settings.GeneralSettings{})
	require.ErrorContains(t, err, "access check failed")
	_, err = deny.SecurityFind(ctx, e.session(), cxrRepoRef)
	require.ErrorContains(t, err, "access check failed")
	_, err = deny.SecurityUpdate(ctx, e.session(), cxrRepoRef, &settings.SecuritySettings{})
	require.ErrorContains(t, err, "access check failed")

	// General settings stay accessible while a repository is migrating, security settings do not.
	_, err = e.db.ExecContext(ctx, "UPDATE repositories SET repo_state = $1 WHERE repo_id = $2",
		int(enum.RepoStateMigrateGitPush), e.repo.ID)
	require.NoError(t, err)
	_, err = allow.GeneralFind(ctx, e.session(), cxrRepoRef)
	require.NoError(t, err)
	_, err = allow.GeneralUpdate(ctx, e.session(), cxrRepoRef, &settings.GeneralSettings{FileSizeLimit: ptr.Int64(5)})
	require.NoError(t, err)
	_, err = allow.SecurityFind(ctx, e.session(), cxrRepoRef)
	require.Error(t, err)
	_, err = allow.SecurityUpdate(ctx, e.session(), cxrRepoRef, &settings.SecuritySettings{})
	require.Error(t, err)
}

func TestCxrSpaceSettingsGeneral(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	root, err := e.spaces.FindByRef(ctx, "root")
	require.NoError(t, err)
	e.createChildSpace(ctx, t, root, "child")
	c := e.controller(cxrAllow)

	// Nothing configured: the global default applies and no scope is reported.
	out, err := c.GeneralFindSpace(ctx, e.session(), "root/child", true)
	require.NoError(t, err)
	require.Equal(t, settings.DefaultBranch, *out.DefaultBranch)
	require.Nil(t, out.DefaultBranchScope)

	_, err = c.GeneralUpdateSpace(ctx, e.session(), "root", &settings.GeneralSettingsSpace{
		DefaultBranch: ptr.String("trunk"),
	})
	require.NoError(t, err)
	require.Equal(t, []audit.Action{audit.ActionUpdated}, e.audit.actions)

	// The child inherits the value and reports where it comes from.
	out, err = c.GeneralFindSpace(ctx, e.session(), "root/child", true)
	require.NoError(t, err)
	require.Equal(t, "trunk", *out.DefaultBranch)
	require.Equal(t, "root", *out.DefaultBranchScope)

	// Without inheritance only the space local value is returned.
	out, err = c.GeneralFindSpace(ctx, e.session(), "root/child", false)
	require.NoError(t, err)
	require.Equal(t, settings.DefaultBranch, *out.DefaultBranch)
	out, err = c.GeneralFindSpace(ctx, e.session(), "root", false)
	require.NoError(t, err)
	require.Equal(t, "trunk", *out.DefaultBranch)

	// Failing audit logs are tolerated.
	e.audit.err = errCxrAuthz
	_, err = c.GeneralUpdateSpace(ctx, e.session(), "root", &settings.GeneralSettingsSpace{
		DefaultBranch: ptr.String("main"),
	})
	require.NoError(t, err)
}

func TestCxrSpaceSettingsErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)
	allow, deny := e.controller(cxrAllow), e.controller(cxrDeny)

	_, err := allow.GeneralFindSpace(ctx, e.session(), "nope", false)
	require.ErrorContains(t, err, "parent space not found")
	_, err = deny.GeneralFindSpace(ctx, e.session(), "root", false)
	require.ErrorContains(t, err, "auth check failed")
	_, err = deny.GeneralUpdateSpace(ctx, e.session(), "root", &settings.GeneralSettingsSpace{})
	require.ErrorContains(t, err, "auth check failed")
	_, err = allow.GeneralUpdateSpace(ctx, e.session(), "root", &settings.GeneralSettingsSpace{
		DefaultBranch: ptr.String("invalid branch name"),
	})
	require.Error(t, err)
}
