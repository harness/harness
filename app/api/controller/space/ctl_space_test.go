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

package space

import (
	"context"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCtlSpaceController_CreateFindUpdate(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	out, err := e.ctrl.Create(ctx, s, &CreateInput{Identifier: "top", Description: " desc "})
	require.NoError(t, err)
	require.Equal(t, "top", out.Identifier)
	require.Equal(t, "desc", out.Description)
	require.False(t, out.IsPublic)
	require.Equal(t, out.ID, out.RootSpaceID)

	// The creator is made owner of the new root space.
	members, count, err := e.ctrl.MembershipList(ctx, s, "top", types.MembershipUserFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, enum.MembershipRoleSpaceOwner, members[0].Role)

	// Child space (nested spaces enabled).
	child, err := e.ctrl.Create(ctx, s, &CreateInput{ParentRef: "top", UID: "child", IsPublic: true})
	require.NoError(t, err)
	require.Equal(t, "top/child", child.Path)
	require.True(t, child.IsPublic)
	require.Equal(t, out.ID, child.RootSpaceID)

	found, err := e.ctrl.Find(ctx, s, "top/child")
	require.NoError(t, err)
	require.Equal(t, child.ID, found.ID)
	require.True(t, found.IsPublic)

	// Update.
	desc := "  new description "
	updated, err := e.ctrl.Update(ctx, s, "top/child", &UpdateInput{Description: &desc})
	require.NoError(t, err)
	require.Equal(t, "new description", updated.Description)

	// No-op update.
	same := "new description"
	_, err = e.ctrl.Update(ctx, s, "top/child", &UpdateInput{Description: &same})
	require.NoError(t, err)
	_, err = e.ctrl.Update(ctx, s, "top/child", &UpdateInput{})
	require.NoError(t, err)

	_, err = e.ctrl.Find(ctx, s, "top/unknown")
	require.Error(t, err)
	_, err = e.ctrl.Update(ctx, s, "top/unknown", &UpdateInput{})
	require.Error(t, err)

	// Children listing.
	spaces, count, err := e.ctrl.ListSpaces(ctx, s, "top", &types.SpaceFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, spaces, 1)
	require.True(t, spaces[0].IsPublic)
}

func TestCtlSpaceController_CreateValidation(t *testing.T) {
	ctx := context.Background()
	s := ctlNewEnv(t, false).session()

	notNested := ctlNewEnv(t, false)
	_, err := notNested.ctrl.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "child"})
	require.ErrorIs(t, err, errNestedSpacesNotSupported)

	e := ctlNewEnv(t, true)
	s = e.session()

	for name, in := range map[string]*CreateInput{
		"negative parent":     {ParentRef: "-1", Identifier: "x"},
		"bad identifier":      {Identifier: ""},
		"illegal root name":   {Identifier: "api"},
		"too long desc":       {Identifier: "x", Description: string(make([]byte, 600))},
		"unknown parent":      {ParentRef: "nope", Identifier: "x"},
		"duplicate root name": {Identifier: "root"},
	} {
		_, err = e.ctrl.Create(ctx, s, in)
		require.Error(t, err, name)
	}

	// Public spaces must be supported by the parent.
	e.publicAccess.unsupported = true
	_, err = e.ctrl.Create(ctx, s, &CreateInput{Identifier: "pub", IsPublic: true})
	require.ErrorIs(t, err, errPublicSpaceCreationDisabled)
	e.publicAccess.unsupported = false

	e.publicAccess.supportErr = errCtlPublicAccess
	_, err = e.ctrl.Create(ctx, s, &CreateInput{Identifier: "pub2"})
	require.ErrorIs(t, err, errCtlPublicAccess)
	e.publicAccess.supportErr = nil

	// Anonymous users cannot create top level spaces.
	anon := &auth.Session{Principal: types.Principal{UID: types.AnonymousPrincipalUID}}
	_, err = e.ctrl.Create(ctx, anon, &CreateInput{Identifier: "anon"})
	require.ErrorIs(t, err, usererror.ErrUnauthorized)
}

func TestCtlSpaceController_AccessFailures(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t, true)
	s := allow.session()
	filter := types.ListQueryFilter{}

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		errs := map[string]error{}
		_, errs["create child"] = c.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "c"})
		_, errs["find"] = c.Find(ctx, s, "root")
		_, errs["update"] = c.Update(ctx, s, "root", &UpdateInput{})
		_, errs["public access"] = c.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{})
		_, errs["move"] = c.Move(ctx, s, "root", &MoveInput{})
		_, errs["soft delete"] = c.SoftDelete(ctx, s, "root")
		_, _, errs["list spaces"] = c.ListSpaces(ctx, s, "root", &types.SpaceFilter{})
		_, _, errs["list repos"] = c.ListRepositories(ctx, s, "root", &types.RepoFilter{})
		_, _, errs["list pipelines"] = c.ListPipelines(ctx, s, "root", types.ListPipelinesFilter{})
		_, _, errs["list executions"] = c.ListExecutions(ctx, s, "root", types.ListExecutionsFilter{})
		_, _, errs["list secrets"] = c.ListSecrets(ctx, s, "root", filter)
		_, _, errs["list templates"] = c.ListTemplates(ctx, s, "root", filter)
		_, _, errs["list connectors"] = c.ListConnectors(ctx, s, "root", filter)
		_, _, errs["list service accounts"] = c.ListServiceAccounts(ctx, s, "root", false, &types.PrincipalFilter{})
		_, errs["membership add"] = c.MembershipAdd(ctx, s, "root", &MembershipAddInput{})
		errs["membership delete"] = c.MembershipDelete(ctx, s, "root", "admin")
		_, errs["membership update"] = c.MembershipUpdate(ctx, s, "root", "admin", &MembershipUpdateInput{})
		_, _, errs["membership list"] = c.MembershipList(ctx, s, "root", types.MembershipUserFilter{})
		_, errs["usage"] = c.GetUsageMetrics(ctx, s, "root", 0, 1)
		_, errs["autolink create"] = c.AutolinkCreate(ctx, s, "root", &autolink.AutoLinkInput{})
		_, _, errs["autolink list"] = c.AutolinkList(ctx, s, "root", &types.AutoLinkFilter{})
		_, errs["autolink find"] = c.AutolinkFind(ctx, s, "root", 1)
		_, errs["autolink update"] = c.AutolinkUpdate(ctx, s, "root", 1, &autolink.AutoLinkUpdateInput{})
		errs["autolink delete"] = c.AutolinkDelete(ctx, s, "root", 1)
		for name, err := range errs {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			} else {
				require.ErrorIs(t, err, apiauth.ErrForbidden, name)
			}
		}
	}
}

func TestCtlSpaceController_UnknownSpace(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()
	filter := types.ListQueryFilter{}

	errs := map[string]error{}
	_, errs["public access"] = e.ctrl.UpdatePublicAccess(ctx, s, "nope", &UpdatePublicAccessInput{})
	_, errs["move"] = e.ctrl.Move(ctx, s, "nope", &MoveInput{})
	_, errs["soft delete"] = e.ctrl.SoftDelete(ctx, s, "nope")
	_, _, errs["list spaces"] = e.ctrl.ListSpaces(ctx, s, "nope", &types.SpaceFilter{})
	_, _, errs["list repos"] = e.ctrl.ListRepositories(ctx, s, "nope", &types.RepoFilter{})
	_, _, errs["list pipelines"] = e.ctrl.ListPipelines(ctx, s, "nope", types.ListPipelinesFilter{})
	_, _, errs["list executions"] = e.ctrl.ListExecutions(ctx, s, "nope", types.ListExecutionsFilter{})
	_, _, errs["list secrets"] = e.ctrl.ListSecrets(ctx, s, "nope", filter)
	_, _, errs["list templates"] = e.ctrl.ListTemplates(ctx, s, "nope", filter)
	_, _, errs["list connectors"] = e.ctrl.ListConnectors(ctx, s, "nope", filter)
	_, _, errs["list service accounts"] = e.ctrl.ListServiceAccounts(ctx, s, "nope", false, &types.PrincipalFilter{})
	_, errs["membership add"] = e.ctrl.MembershipAdd(ctx, s, "nope", &MembershipAddInput{})
	errs["membership delete"] = e.ctrl.MembershipDelete(ctx, s, "nope", "admin")
	_, errs["membership update"] = e.ctrl.MembershipUpdate(ctx, s, "nope", "admin", &MembershipUpdateInput{})
	_, _, errs["membership list"] = e.ctrl.MembershipList(ctx, s, "nope", types.MembershipUserFilter{})
	_, errs["usage"] = e.ctrl.GetUsageMetrics(ctx, s, "nope", 0, 1)
	_, errs["autolink find"] = e.ctrl.AutolinkFind(ctx, s, "nope", 1)
	for name, err := range errs {
		require.Error(t, err, name)
	}
}

func TestCtlSpaceController_UpdatePublicAccess(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	out, err := e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{IsPublic: true})
	require.NoError(t, err)
	require.True(t, out.IsPublic)

	// No-op.
	out, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{IsPublic: true})
	require.NoError(t, err)
	require.True(t, out.IsPublic)

	out, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{IsPublic: false})
	require.NoError(t, err)
	require.False(t, out.IsPublic)

	e.publicAccess.unsupported = true
	_, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{IsPublic: true})
	require.ErrorIs(t, err, errPublicSpaceCreationDisabled)
	e.publicAccess.unsupported = false

	e.publicAccess.supportErr = errCtlPublicAccess
	_, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{})
	require.ErrorIs(t, err, errCtlPublicAccess)
	e.publicAccess.supportErr = nil

	e.publicAccess.getErr = errCtlPublicAccess
	_, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{})
	require.ErrorIs(t, err, errCtlPublicAccess)
	_, err = e.ctrl.Find(ctx, s, "root")
	require.ErrorIs(t, err, errCtlPublicAccess)
	e.publicAccess.getErr = nil

	e.publicAccess.setErr = errCtlPublicAccess
	_, err = e.ctrl.UpdatePublicAccess(ctx, s, "root", &UpdatePublicAccessInput{IsPublic: true})
	require.ErrorIs(t, err, errCtlPublicAccess)
}

func TestCtlSpaceController_Membership(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()
	e.createUser(t, "member", false)

	for name, in := range map[string]*MembershipAddInput{
		"no user":  {Role: enum.MembershipRoleReader},
		"no role":  {UserUID: "member"},
		"bad role": {UserUID: "member", Role: "nope"},
		"unknown":  {UserUID: "ghost", Role: enum.MembershipRoleReader},
	} {
		_, err := e.ctrl.MembershipAdd(ctx, s, "root", in)
		require.Error(t, err, name)
	}

	added, err := e.ctrl.MembershipAdd(ctx, s, "root", &MembershipAddInput{
		UserUID: "member", Role: enum.MembershipRoleReader,
	})
	require.NoError(t, err)
	require.Equal(t, enum.MembershipRoleReader, added.Role)
	require.Equal(t, "member", added.Principal.UID)

	// Adding twice conflicts.
	_, err = e.ctrl.MembershipAdd(ctx, s, "root", &MembershipAddInput{
		UserUID: "member", Role: enum.MembershipRoleReader,
	})
	require.Error(t, err)

	members, count, err := e.ctrl.MembershipList(ctx, s, "root", types.MembershipUserFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, members, 1)

	// Full first page triggers the count query.
	_, count, err = e.ctrl.MembershipList(ctx, s, "root", types.MembershipUserFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 1}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	for name, in := range map[string]*MembershipUpdateInput{
		"no role":  {},
		"bad role": {Role: "nope"},
	} {
		_, err = e.ctrl.MembershipUpdate(ctx, s, "root", "member", in)
		require.Error(t, err, name)
	}
	_, err = e.ctrl.MembershipUpdate(ctx, s, "root", "ghost", &MembershipUpdateInput{Role: enum.MembershipRoleContributor})
	require.Error(t, err)

	// Update of a non-member fails.
	e.createUser(t, "outsider", false)
	_, err = e.ctrl.MembershipUpdate(
		ctx, s, "root", "outsider", &MembershipUpdateInput{Role: enum.MembershipRoleContributor})
	require.Error(t, err)

	updated, err := e.ctrl.MembershipUpdate(ctx, s, "root", "member", &MembershipUpdateInput{
		Role: enum.MembershipRoleContributor,
	})
	require.NoError(t, err)
	require.Equal(t, enum.MembershipRoleContributor, updated.Role)

	// Same role is a no-op.
	_, err = e.ctrl.MembershipUpdate(ctx, s, "root", "member", &MembershipUpdateInput{
		Role: enum.MembershipRoleContributor,
	})
	require.NoError(t, err)

	require.NoError(t, e.ctrl.MembershipDelete(ctx, s, "root", "member"))
	require.Error(t, e.ctrl.MembershipDelete(ctx, s, "root", "ghost"))
}

func TestCtlSpaceController_ListChildren(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	repo := e.createRepo(t, e.root, "repo1")
	e.createRepo(t, e.root, "repo2")

	// Empty space.
	empty := e.createSpace(t, "empty", 0)
	repos, count, err := e.ctrl.ListRepositories(ctx, s, empty.Identifier, &types.RepoFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, repos)

	repos, count, err = e.ctrl.ListRepositories(ctx, s, "root", &types.RepoFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Len(t, repos, 2)
	require.Contains(t, repos[0].GitURL, "https://git.example.com/root/")
	require.False(t, repos[0].IsFavorite)

	// Favorites.
	fav := e.ctrl.favoriteStore
	require.NoError(t, fav.Create(ctx, e.admin.ID, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo}))
	repos, _, err = e.ctrl.ListRepositories(ctx, s, "root", &types.RepoFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	favorites := 0
	for _, r := range repos {
		if r.IsFavorite {
			favorites++
		}
	}
	require.Equal(t, 1, favorites)

	onlyFavs := e.admin.ID
	repos, count, err = e.ctrl.ListRepositories(ctx, s, "root", &types.RepoFilter{
		Page: 1, Size: 10, OnlyFavoritesFor: &onlyFavs,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.True(t, repos[0].IsFavorite)
}

func TestCtlSpaceController_ListServiceAccounts(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	child := e.createSpace(t, "child", e.root.ID)
	sa := &types.ServiceAccount{
		UID: "sa1", Email: "sa1@example.com", DisplayName: "SA", Salt: "salt",
		ParentType: enum.ParentResourceTypeSpace, ParentID: e.root.ID,
	}
	require.NoError(t, e.principals.CreateServiceAccount(ctx, sa))

	filter := &types.PrincipalFilter{Page: 1, Size: 10}
	accounts, count, err := e.ctrl.ListServiceAccounts(ctx, s, "root", false, filter)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, accounts, 1)
	require.Equal(t, "sa1", accounts[0].UID)

	// Inherited accounts of the child space come from the parent.
	accounts, count, err = e.ctrl.ListServiceAccounts(ctx, s, "root/"+child.Identifier, true, filter)
	require.NoError(t, err)
	require.NotNil(t, accounts)
	require.GreaterOrEqual(t, count, int64(0))
}

func TestCtlSpaceController_ListResources(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	pipelines, count, err := e.ctrl.ListPipelines(ctx, s, "root", types.ListPipelinesFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, pipelines)

	execs, count, err := e.ctrl.ListExecutions(ctx, s, "root", types.ListExecutionsFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		Sort:            enum.ExecutionSortStarted,
		Order:           enum.OrderDesc,
	})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, execs)

	// A full page triggers the count queries.
	_, count, err = e.ctrl.ListExecutions(ctx, s, "root", types.ListExecutionsFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 10}},
		Sort:            enum.ExecutionSortStarted,
		Order:           enum.OrderDesc,
	})
	require.NoError(t, err)
	require.Zero(t, count)
	_, count, err = e.ctrl.ListPipelines(ctx, s, "root", types.ListPipelinesFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 10}},
	})
	require.NoError(t, err)
	require.Zero(t, count)

	filter := types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}
	secrets, count, err := e.ctrl.ListSecrets(ctx, s, "root", filter)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, secrets)

	templates, count, err := e.ctrl.ListTemplates(ctx, s, "root", filter)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, templates)

	connectors, count, err := e.ctrl.ListConnectors(ctx, s, "root", filter)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, connectors)
}

func TestCtlSpaceController_SoftDelete(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	space := e.createSpace(t, "doomed", 0)
	e.createRepo(t, space, "repo1")
	e.publicAccess.public[space.Identifier] = true

	resp, err := e.ctrl.SoftDelete(ctx, s, "doomed")
	require.NoError(t, err)
	require.NotZero(t, resp.DeletedAt)
	require.NotContains(t, e.publicAccess.public, space.Identifier)

	// A soft deleted space without repositories can be purged.
	require.NoError(t, e.ctrl.Purge(ctx, s, "doomed", resp.DeletedAt))
	require.Error(t, e.ctrl.Purge(ctx, s, "doomed", resp.DeletedAt))
}

func TestCtlSpaceController_Autolinks(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	created, err := e.ctrl.AutolinkCreate(ctx, s, "root", &autolink.AutoLinkInput{
		Type: enum.AutoLinkTypeRegex, Pattern: `JIRA-(\d+)`, TargetURL: "https://jira.example.com/<num>",
	})
	require.NoError(t, err)

	found, err := e.ctrl.AutolinkFind(ctx, s, "root", created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)

	list, count, err := e.ctrl.AutolinkList(ctx, s, "root", &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	pattern := `BUG-(\d+)`
	updated, err := e.ctrl.AutolinkUpdate(ctx, s, "root", created.ID, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
	require.NoError(t, err)
	require.Equal(t, pattern, updated.Pattern)

	_, err = e.ctrl.AutolinkCreate(ctx, s, "root", &autolink.AutoLinkInput{})
	require.Error(t, err)
	_, err = e.ctrl.AutolinkUpdate(ctx, s, "root", created.ID+100, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
	require.Error(t, err)
	_, err = e.ctrl.AutolinkFind(ctx, s, "root", created.ID+100)
	require.Error(t, err)

	require.NoError(t, e.ctrl.AutolinkDelete(ctx, s, "root", created.ID))
	require.Error(t, e.ctrl.AutolinkDelete(ctx, s, "root", created.ID))
}

func TestCtlSpaceController_UsageMetrics(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	// Only root spaces are accepted.
	_, err := e.ctrl.GetUsageMetrics(ctx, s, "root/sub", 0, 1)
	require.Error(t, err)
}

func TestCtlSpaceController_Move(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t, true)
	s := e.session()

	child := e.createSpace(t, "child", e.root.ID)
	same := child.Identifier
	// Nothing changes: returns the space as is.
	out, err := e.ctrl.Move(ctx, s, "root/child", &MoveInput{UID: &same})
	require.NoError(t, err)
	require.Equal(t, child.ID, out.ID)

	bad := ""
	_, err = e.ctrl.Move(ctx, s, "root/child", &MoveInput{Identifier: &bad})
	require.Error(t, err)

	unknown := "nope"
	_, err = e.ctrl.Move(ctx, s, "root/child", &MoveInput{ParentRef: &unknown})
	require.Error(t, err)
}
