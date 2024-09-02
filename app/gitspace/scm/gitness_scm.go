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

package scm

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/token"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ Provider = (*GitnessSCM)(nil)

var gitspaceJWTLifetime = 720 * 24 * time.Hour

const defaultGitspacePATIdentifier = "Gitspace_Default"

type GitnessSCM struct {
	git            git.Interface
	repoStore      store.RepoStore
	tokenStore     store.TokenStore
	principalStore store.PrincipalStore
	urlProvider    urlprovider.Provider
}

// ListBranches implements Provider.
func (s *GitnessSCM) ListBranches(ctx context.Context,
	filter *BranchFilter,
	_ *ResolvedCredentials) ([]Branch, error) {
	repo, err := s.repoStore.FindByRef(ctx, filter.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}
	rpcOut, err := s.git.ListBranches(ctx, &git.ListBranchesParams{
		ReadParams:    git.CreateReadParams(repo),
		IncludeCommit: false,
		Query:         filter.Query,
		Sort:          git.BranchSortOptionDate,
		Order:         git.SortOrderDesc,
		Page:          int32(filter.Page),
		PageSize:      int32(filter.Size),
	})
	if err != nil {
		return nil, err
	}
	branches := make([]Branch, len(rpcOut.Branches))
	for i := range rpcOut.Branches {
		branches[i] = mapBranch(rpcOut.Branches[i])
	}

	return branches, nil
}

func mapBranch(b git.Branch) Branch {
	return Branch{
		Name: b.Name,
		SHA:  b.SHA.String(),
	}
}

// ListReporisotries implements Provider.
func (s *GitnessSCM) ListReporisotries(ctx context.Context,
	filter *RepositoryFilter,
	_ *ResolvedCredentials) ([]Repository, error) {
	repos, err := s.repoStore.List(ctx, filter.SpaceID, &types.RepoFilter{
		Page:  filter.Page,
		Size:  filter.Size,
		Query: filter.Query,
		Sort:  enum.RepoAttrUpdated,
		Order: enum.OrderDesc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list child repos: %w", err)
	}
	var reposOut []Repository
	for _, repo := range repos {
		// backfill URLs
		repo.GitURL = s.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
		repo.GitSSHURL = s.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

		repoOut, err := mapRepository(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo %q output: %w", repo.Path, err)
		}

		reposOut = append(reposOut, repoOut)
	}
	return reposOut, nil
}

func mapRepository(repo *types.Repository) (Repository, error) {
	if repo == nil {
		return Repository{}, fmt.Errorf("repository is null")
	}
	return Repository{
		Name:          repo.Identifier,
		DefaultBranch: repo.DefaultBranch,
		GitURL:        repo.GitURL,
		GitSSHURL:     repo.GitSSHURL,
	}, nil
}

func NewGitnessSCM(repoStore store.RepoStore, git git.Interface,
	tokenStore store.TokenStore,
	principalStore store.PrincipalStore,
	urlProvider urlprovider.Provider) *GitnessSCM {
	return &GitnessSCM{
		repoStore:      repoStore,
		git:            git,
		tokenStore:     tokenStore,
		principalStore: principalStore,
		urlProvider:    urlProvider,
	}
}

func (s GitnessSCM) ResolveCredentials(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (*ResolvedCredentials, error) {
	repoURL, err := url.Parse(gitspaceConfig.CodeRepo.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL %s: %w", gitspaceConfig.CodeRepo.URL, err)
	}
	repoName := strings.TrimSuffix(path.Base(repoURL.Path), ".git")
	repo, err := s.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepo.Ref)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}
	cloneURL, err := url.Parse(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clone url '%s': %w", cloneURL, err)
	}
	// Backfill clone URL
	gitURL := s.urlProvider.GenerateContainerGITCloneURL(ctx, repo.Path)
	resolvedCredentails := &ResolvedCredentials{Branch: gitspaceConfig.CodeRepo.Branch, CloneURL: gitURL}
	resolvedCredentails.RepoName = repoName
	gitspacePrincipal := bootstrap.NewGitspaceServiceSession().Principal
	user, err := findUserFromUID(ctx, s.principalStore, gitspaceConfig.GitspaceUser.Identifier)
	if err != nil {
		return nil, err
	}
	var jwtToken string
	existingToken, _ := s.tokenStore.FindByIdentifier(ctx, user.ID, defaultGitspacePATIdentifier)
	if existingToken != nil {
		// create jwt token.
		jwtToken, err = jwt.GenerateForToken(existingToken, user.ToPrincipal().Salt)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWT token: %w", err)
		}
	} else {
		_, jwtToken, err = token.CreatePAT(
			ctx,
			s.tokenStore,
			&gitspacePrincipal,
			user,
			defaultGitspacePATIdentifier,
			&gitspaceJWTLifetime)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT: %w", err)
	}
	modifiedURL, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("error while parsing the clone url: %s", gitURL)
	}
	credentials := &Credentials{
		Email:    user.Email,
		Name:     user.DisplayName,
		Password: jwtToken,
		Host:     modifiedURL.Host,
		Protocol: modifiedURL.Scheme,
		Path:     modifiedURL.Path,
	}
	resolvedCredentails.Credentials = credentials
	return resolvedCredentails, nil
}

func (s GitnessSCM) GetFileContent(ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	filePath string,
	_ *ResolvedCredentials,
) ([]byte, error) {
	repo, err := s.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepo.Ref)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}
	// create read params once
	readParams := git.CreateReadParams(repo)
	treeNodeOutput, err := s.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitspaceConfig.CodeRepo.Branch,
		Path:                filePath,
		IncludeLatestCommit: false,
	})
	if err != nil {
		return make([]byte, 0), nil //nolint:nilerr
	}

	// viewing Raw content is only supported for blob content
	if treeNodeOutput.Node.Type != git.TreeNodeTypeBlob {
		return nil, usererror.BadRequestf(
			"Object in '%s' at '/%s' is of type '%s'. Only objects of type %s support raw viewing.",
			gitspaceConfig.CodeRepo.Branch, filePath, treeNodeOutput.Node.Type, git.TreeNodeTypeBlob)
	}

	blobReader, err := s.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        treeNodeOutput.Node.SHA,
		SizeLimit:  0, // no size limit, we stream whatever data there is
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}
	catFileOutput, err := io.ReadAll(blobReader.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}
	return catFileOutput, nil
}

func findUserFromUID(ctx context.Context,
	principalStore store.PrincipalStore, userUID string,
) (*types.User, error) {
	return principalStore.FindUserByUID(ctx, userUID)
}
