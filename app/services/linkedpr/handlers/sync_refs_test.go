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

package handlers

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	gitpkg "github.com/harness/gitness/git"
	mockgit "github.com/harness/gitness/mocks/git"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	_ "unsafe" // for go:linkname
)

//go:linkname bootstrapSystemServicePrincipal github.com/harness/gitness/app/bootstrap.systemServicePrincipal
var bootstrapSystemServicePrincipal *types.Principal

func init() {
	bootstrapSystemServicePrincipal = &types.Principal{
		ID:          1,
		UID:         "harness-test",
		DisplayName: "Harness Test",
		Email:       "harness-test@local",
	}
}

type recordingConnectorService struct {
	lastDef importer.ConnectorDef
}

func (s *recordingConnectorService) GetAccessInfo(
	_ context.Context, def importer.ConnectorDef,
) (importer.AccessInfo, error) {
	s.lastDef = def
	return importer.AccessInfo{URL: "https://github.com/acme/widget.git"}, nil
}

func (s *recordingConnectorService) FetchProviderRepoInfo(
	_ context.Context, _ importer.ConnectorDef,
) (importer.ProviderRepoInfo, error) {
	return importer.ProviderRepoInfo{}, nil
}

func (s *recordingConnectorService) ResolveConnectorRef(_, ref string) (string, string) {
	return "", ref
}
func (s *recordingConnectorService) EncodeConnectorRef(_, _, identifier string) string {
	return identifier
}

type testURLProvider struct{}

func (testURLProvider) GetInternalAPIURL(context.Context) string {
	return "http://localhost:3000/api"
}
func (testURLProvider) GenerateContainerGITCloneURL(context.Context, string) string { return "" }
func (testURLProvider) GenerateGITCloneURL(context.Context, string) string          { return "" }
func (testURLProvider) GenerateGITCloneSSHURL(context.Context, string) string       { return "" }
func (testURLProvider) GenerateUIRepoURL(context.Context, string) string            { return "" }
func (testURLProvider) GenerateUIPRURL(context.Context, string, int64) string       { return "" }
func (testURLProvider) GenerateUICompareURL(context.Context, string, string, string) string {
	return ""
}
func (testURLProvider) GenerateUIRefURL(context.Context, string, string) string { return "" }
func (testURLProvider) GetAPIHostname(context.Context) string                   { return "localhost" }
func (testURLProvider) GenerateUIBuildURL(context.Context, string, string, int64) string {
	return ""
}
func (testURLProvider) GetGITHostname(context.Context) string { return "localhost" }
func (testURLProvider) GetAPIProto(context.Context) string    { return "http" }
func (testURLProvider) RegistryURL(context.Context, ...string) string {
	return ""
}
func (testURLProvider) PackageURL(context.Context, string, string, ...string) string { return "" }
func (testURLProvider) GetUIBaseURL(context.Context, ...string) string               { return "" }
func (testURLProvider) PackagePathFor(context.Context, gitnessurl.PackagePathSpec) (string, error) {
	return "", nil
}
func (testURLProvider) GenerateUIRegistryURL(context.Context, string, string) string { return "" }

type repoIDCacheStub struct {
	repo *types.RepositoryCore
}

func (s *repoIDCacheStub) Stats() (int64, int64)            { return 0, 0 }
func (s *repoIDCacheStub) Evict(_ context.Context, _ int64) {}
func (s *repoIDCacheStub) Get(_ context.Context, _ int64) (*types.RepositoryCore, error) {
	return s.repo, nil
}

type spacePathCacheStub struct{}

func (s *spacePathCacheStub) Stats() (int64, int64)             { return 0, 0 }
func (s *spacePathCacheStub) Evict(_ context.Context, _ string) {}
func (s *spacePathCacheStub) Get(_ context.Context, _ string) (*types.SpacePath, error) {
	return nil, gitness_store.ErrResourceNotFound
}

type repoRefCacheStub struct{}

func (s *repoRefCacheStub) Stats() (int64, int64)                         { return 0, 0 }
func (s *repoRefCacheStub) Evict(_ context.Context, _ types.RepoCacheKey) {}
func (s *repoRefCacheStub) Get(_ context.Context, _ types.RepoCacheKey) (int64, error) {
	return 0, nil
}

func TestRunSyncRefs_PassesConnectorRepoToGetAccessInfo(t *testing.T) {
	t.Parallel()

	connectorSvc := &recordingConnectorService{}
	gitClient := &mockgit.Interface{}
	gitClient.On("SyncRefs", mock.Anything, mock.Anything).Return(&gitpkg.SyncRefsOutput{}, nil)

	repoFinder := refcache.NewRepoFinder(
		nil,
		&spacePathCacheStub{},
		&repoIDCacheStub{repo: &types.RepositoryCore{ID: 1, GitUID: "git-uid-1"}},
		&repoRefCacheStub{},
		storecache.Evictor[*types.RepositoryCore]{},
	)

	linkedRepo := &types.LinkedRepo{
		RepoID:              1,
		ConnectorPath:       "MU-account",
		ConnectorIdentifier: "github_account_level_connector",
		ConnectorRepo:       "personalTest",
	}

	err := RunSyncRefs(
		t.Context(),
		gitClient,
		repoFinder,
		testURLProvider{},
		connectorSvc,
		linkedRepo,
		[]string{"refs/pull/8/head"},
	)
	require.NoError(t, err)
	assert.Equal(t, importer.ConnectorDef{
		Path:           "MU-account",
		Identifier:     "github_account_level_connector",
		RepoIdentifier: "personalTest",
	}, connectorSvc.lastDef)
}
