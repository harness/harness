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

package repo

import (
	"context"
	"errors"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errCtlGit = errors.New("git failure")

type ctlIndexer struct{ indexed []int64 }

func (i *ctlIndexer) Index(_ context.Context, repo *types.Repository) error {
	i.indexed = append(i.indexed, repo.ID)
	return nil
}

func ctlGitCreateOK(e *ctlEnv) {
	e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(
		func(_ context.Context, p *git.CreateRepositoryParams) (*git.CreateRepositoryOutput, error) {
			return &git.CreateRepositoryOutput{UID: "git-" + p.DefaultBranch + "-uid"}, nil
		})
}

func TestCtlRepoController_Create(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	indexer := &ctlIndexer{}
	e.ctrl.indexer = indexer
	ctlGitCreateOK(e)

	out, err := e.ctrl.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "new-repo", Description: " desc "})
	require.NoError(t, err)
	require.Equal(t, "new-repo", out.Identifier)
	require.Equal(t, "desc", out.Description)
	require.Equal(t, "main", out.DefaultBranch)
	require.True(t, out.IsEmpty)
	require.False(t, out.IsPublic)
	require.Equal(t, "https://git.example.com/root/new-repo.git", out.GitURL)
	require.Empty(t, indexer.indexed)

	// With files the repository is not empty and gets indexed.
	out, err = e.ctrl.Create(ctx, s, &CreateInput{
		ParentRef: "root", UID: "with-files", DefaultBranch: "trunk", IsPublic: true,
		CreateFileOptions: CreateFileOptions{Readme: true, License: "apache-2.0", GitIgnore: "Go"},
	})
	require.NoError(t, err)
	require.False(t, out.IsEmpty)
	require.True(t, out.IsPublic)
	require.Equal(t, "trunk", out.DefaultBranch)
	require.Equal(t, []int64{out.ID}, indexer.indexed)

	// Duplicates are conflicts.
	_, err = e.ctrl.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "new-repo"})
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 409, userErr.Status)
}

func TestCtlRepoController_CreateErrors(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	for name, in := range map[string]*CreateInput{
		"no parent":       {Identifier: "x"},
		"bad identifier":  {ParentRef: "root", Identifier: ""},
		"illegal suffix":  {ParentRef: "root", Identifier: "x.git"},
		"unknown parent":  {ParentRef: "missing", Identifier: "x"},
		"long desc":       {ParentRef: "root", Identifier: "x", Description: string(make([]byte, 1000))},
		"bad license":     {ParentRef: "root", Identifier: "x", CreateFileOptions: CreateFileOptions{License: "nope"}},
		"bad gitignore":   {ParentRef: "root", Identifier: "x", CreateFileOptions: CreateFileOptions{GitIgnore: "nope"}},
		"negative parent": {ParentRef: "-5", Identifier: "x"},
	} {
		ctlGitCreateOK(e)
		_, err := e.ctrl.Create(ctx, s, in)
		require.Error(t, err, name)
	}

	// Git failure.
	e.git.ExpectedCalls = nil
	e.git.On("CreateRepository", mock.Anything, mock.Anything).Return(nil, errCtlGit)
	_, err := e.ctrl.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "gitfail"})
	require.ErrorIs(t, err, errCtlGit)

	// The failed repo is not stored.
	_, err = e.ctrl.Find(ctx, s, "root/gitfail")
	require.Error(t, err)

	// Access denied.
	deny := *e.ctrl
	deny.authorizer = ctlAuthorizer{mode: ctlDeny}
	_, err = deny.Create(ctx, s, &CreateInput{ParentRef: "root", Identifier: "denied"})
	require.ErrorIs(t, err, apiauth.ErrForbidden)
}

func TestCtlRepoController_FindAndSummary(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	require.NoError(t, e.ctrl.favoriteStore.Create(ctx, e.admin.ID, &types.FavoriteResource{
		ID: e.repo.ID, Type: enum.ResourceTypeRepo,
	}))

	out, err := e.ctrl.Find(ctx, s, "root/repo1")
	require.NoError(t, err)
	require.Equal(t, e.repo.ID, out.ID)
	require.True(t, out.IsFavorite)
	require.Contains(t, out.GitSSHURL, "ssh://git@git.example.com/root/repo1")

	e.git.On("Summary", mock.Anything, mock.Anything).Return(
		git.SummaryOutput{CommitCount: 7, BranchCount: 2, TagCount: 1}, nil).Once()
	sum, err := e.ctrl.Summary(ctx, s, "root/repo1")
	require.NoError(t, err)
	require.Equal(t, 7, sum.DefaultBranchCommitCount)
	require.Equal(t, 2, sum.BranchCount)
	require.Equal(t, 1, sum.TagCount)

	e.git.On("Summary", mock.Anything, mock.Anything).Return(git.SummaryOutput{}, errCtlGit).Once()
	_, err = e.ctrl.Summary(ctx, s, "root/repo1")
	require.ErrorIs(t, err, errCtlGit)
}

func TestCtlRepoController_ListEndpoints(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	acts, count, err := e.ctrl.ListActivities(ctx, s, "root/repo1", &types.RepoActivityFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, acts)
	_, count, err = e.ctrl.ListActivities(ctx, s, "root/repo1", &types.RepoActivityFilter{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Zero(t, count)

	pipelines, pcount, err := e.ctrl.ListPipelines(ctx, s, "root/repo1", &types.ListPipelinesFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.Zero(t, pcount)
	require.Empty(t, pipelines)
	_, _, err = e.ctrl.ListPipelines(ctx, s, "root/repo1", &types.ListPipelinesFilter{Latest: true})
	require.NoError(t, err)

	sa := &types.ServiceAccount{
		UID: "sa1", Email: "sa1@example.com", DisplayName: "SA", Salt: "salt",
		ParentType: enum.ParentResourceTypeRepo, ParentID: e.repo.ID,
	}
	require.NoError(t, e.principals.CreateServiceAccount(ctx, sa))
	saFilter := &types.PrincipalFilter{Page: 1, Size: 10}
	accounts, acount, err := e.ctrl.ListServiceAccounts(ctx, s, "root/repo1", false, saFilter)
	require.NoError(t, err)
	require.EqualValues(t, 1, acount)
	require.Len(t, accounts, 1)
	_, _, err = e.ctrl.ListServiceAccounts(ctx, s, "root/repo1", true, saFilter)
	require.NoError(t, err)

	langs, err := e.ctrl.GetLanguages(ctx, s, "root/repo1")
	require.NoError(t, err)
	require.Empty(t, langs)
}

func TestCtlRepoController_PublicAccess(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	out, err := e.ctrl.UpdatePublicAccess(ctx, s, "root/repo1", &UpdatePublicAccessInput{IsPublic: true})
	require.NoError(t, err)
	require.True(t, out.IsPublic)

	// No-op and revert.
	out, err = e.ctrl.UpdatePublicAccess(ctx, s, "root/repo1", &UpdatePublicAccessInput{IsPublic: true})
	require.NoError(t, err)
	require.True(t, out.IsPublic)
	out, err = e.ctrl.UpdatePublicAccess(ctx, s, "root/repo1", &UpdatePublicAccessInput{IsPublic: false})
	require.NoError(t, err)
	require.False(t, out.IsPublic)
}

func TestCtlRepoController_SoftDeleteRestorePurge(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

	repo := e.createRepo(t, e.space, "doomed")
	e.publicAccess.public["root/doomed"] = true

	resp, err := e.ctrl.SoftDelete(ctx, s, "root/doomed")
	require.NoError(t, err)
	require.NotZero(t, resp.DeletedAt)
	require.NotContains(t, e.publicAccess.public, "root/doomed")

	// Deleted repos cannot be found any more, nor deleted twice.
	_, err = e.ctrl.Find(ctx, s, "root/doomed")
	require.Error(t, err)
	_, err = e.ctrl.SoftDelete(ctx, s, "root/doomed")
	require.Error(t, err)

	// Restore into a different name.
	newName := "reborn"
	out, err := e.ctrl.Restore(ctx, s, "root/doomed", resp.DeletedAt, &RestoreInput{NewIdentifier: &newName})
	require.NoError(t, err)
	require.Equal(t, repo.ID, out.ID)
	require.Equal(t, "reborn", out.Identifier)
	require.False(t, out.IsPublic)

	// Restoring a repo that is not deleted fails.
	_, err = e.ctrl.Restore(ctx, s, "root/reborn", resp.DeletedAt, &RestoreInput{})
	require.Error(t, err)

	// Restoring into an unknown parent fails.
	resp, err = e.ctrl.SoftDelete(ctx, s, "root/reborn")
	require.NoError(t, err)
	missing := "missing"
	_, err = e.ctrl.Restore(ctx, s, "root/reborn", resp.DeletedAt, &RestoreInput{NewParentRef: &missing})
	require.Error(t, err)

	// Purge.
	require.Error(t, e.ctrl.Purge(ctx, s, "root/reborn", resp.DeletedAt+1))
	require.NoError(t, e.ctrl.Purge(ctx, s, "root/reborn", resp.DeletedAt))
	require.Error(t, e.ctrl.Purge(ctx, s, "root/reborn", resp.DeletedAt))
	e.git.AssertCalled(t, "DeleteRepository", mock.Anything, mock.Anything)

	// The git deletion tolerates not found errors but reports others.
	e.git.ExpectedCalls = nil
	e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(errCtlGit)
	require.ErrorIs(t, e.ctrl.DeleteGitRepository(ctx, s, "uid"), errCtlGit)
}

func TestCtlRepoController_EnsureIdentifierAvailable(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	e.git.On("DeleteRepository", mock.Anything, mock.Anything).Return(nil)

	require.NoError(t, e.ctrl.EnsureIdentifierAvailable(ctx, s, e.space.ID, "free"))

	err := e.ctrl.EnsureIdentifierAvailable(ctx, s, e.space.ID, "repo1")
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 409, userErr.Status)

	// A repository whose import failed is purged to free up the identifier.
	failed := e.createRepo(t, e.space, "failed-import")
	failed.State = enum.RepoStateImportFailed
	require.NoError(t, e.repos.Update(ctx, failed))
	e.repoFinder.MarkChanged(ctx, failed.Core())
	require.NoError(t, e.ctrl.EnsureIdentifierAvailable(ctx, s, e.space.ID, "failed-import"))
	_, err = e.repos.FindActiveByUID(ctx, e.space.ID, "failed-import")
	require.Error(t, err)
}

func TestCtlRepoController_ValidationHelpers(t *testing.T) {
	require.NoError(t, ValidateParentRef("root"))
	require.NoError(t, ValidateParentRef("12"))
	require.Error(t, ValidateParentRef(""))
	require.Error(t, ValidateParentRef("  "))
	require.Error(t, ValidateParentRef("0"))
	require.Error(t, ValidateParentRef("-1"))

	repo := &types.Repository{Identifier: "abc"}
	out := RepositoryOutput{Repository: *repo}
	data, err := out.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"uid":"abc"`)
}
