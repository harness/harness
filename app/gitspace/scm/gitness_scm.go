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
	repoURL, err := url.Parse(gitspaceConfig.CodeRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL %s: %w", gitspaceConfig.CodeRepoURL, err)
	}
	repoName := strings.TrimSuffix(path.Base(repoURL.Path), ".git")
	repo, err := s.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}
	cloneURL, err := url.Parse(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clone url '%s': %w", cloneURL, err)
	}
	// Backfill clone URL
	gitURL := s.urlProvider.GenerateContainerGITCloneURL(ctx, repo.Path)
	resolvedCredentails := &ResolvedCredentials{Branch: gitspaceConfig.Branch, CloneURL: gitURL}
	resolvedCredentails.RepoName = repoName
	gitspacePrincipal := bootstrap.NewGitspaceServiceSession().Principal
	user, err := findUserFromUID(ctx, s.principalStore, gitspaceConfig.UserID)
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
	credentials := &Credentials{
		Password: jwtToken,
		Email:    user.Email,
		Name:     user.DisplayName,
	}
	resolvedCredentails.Credentials = credentials
	return resolvedCredentails, nil
}

func (s GitnessSCM) GetFileContent(ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	filePath string,
) ([]byte, error) {
	repo, err := s.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}
	// create read params once
	readParams := git.CreateReadParams(repo)
	treeNodeOutput, err := s.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitspaceConfig.Branch,
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
			gitspaceConfig.Branch, filePath, treeNodeOutput.Node.Type, git.TreeNodeTypeBlob)
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
