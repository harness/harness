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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/token"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoDefaultBranch = errors.New("no default branch")
)

var gitspaceJWTLifetime = 720 * 24 * time.Hour

const defaultGitspacePATIdentifier = "Gitspace_Default"

var _ SCM = (*genericSCM)(nil)

type SCM interface {
	// RepoNameAndDevcontainerConfig fetches repository name & devcontainer config file from the given repo and branch.
	Resolve(
		ctx context.Context,
		gitspaceConfig *types.GitspaceConfig,
	) (*ResolvedDetails, error)

	// CheckValidCodeRepo checks if the current URL is a valid and accessible code repo,
	// input can be connector info, user token etc.
	CheckValidCodeRepo(ctx context.Context, request CodeRepositoryRequest) (*CodeRepositoryResponse, error)
}

type genericSCM struct {
	git            git.Interface
	repoStore      store.RepoStore
	tokenStore     store.TokenStore
	principalStore store.PrincipalStore
	urlProvider    urlprovider.Provider
}

func NewSCM(repoStore store.RepoStore, git git.Interface,
	tokenStore store.TokenStore,
	principalStore store.PrincipalStore,
	urlProvider urlprovider.Provider) SCM {
	return &genericSCM{
		repoStore:      repoStore,
		git:            git,
		tokenStore:     tokenStore,
		principalStore: principalStore,
		urlProvider:    urlProvider,
	}
}

func (s genericSCM) CheckValidCodeRepo(
	ctx context.Context,
	request CodeRepositoryRequest,
) (*CodeRepositoryResponse, error) {
	err := validateURL(request)
	if err != nil {
		return nil, fmt.Errorf("invalid URL, %w", err)
	}
	codeRepositoryResponse := &CodeRepositoryResponse{
		URL:               request.URL,
		CodeRepoIsPrivate: true,
	}
	defaultBranch, err := detectDefaultGitBranch(ctx, request.URL)
	if err == nil {
		branch := "main"
		if defaultBranch != "" {
			branch = defaultBranch
		}
		codeRepositoryResponse = &CodeRepositoryResponse{
			URL:               request.URL,
			Branch:            branch,
			CodeRepoIsPrivate: false,
		}
	}
	return codeRepositoryResponse, nil
}

func (s genericSCM) Resolve(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*ResolvedDetails, error) {
	resolvedDetails := &ResolvedDetails{Branch: gitspaceConfig.Branch, CloneURL: gitspaceConfig.CodeRepoURL}
	repoURL, err := url.Parse(gitspaceConfig.CodeRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL %s: %w", gitspaceConfig.CodeRepoURL, err)
	}
	repoName := strings.TrimSuffix(path.Base(repoURL.Path), ".git")
	resolvedDetails.RepoName = repoName
	gitWorkingDirectory := "/tmp/git/"

	cloneDir := gitWorkingDirectory + uuid.New().String()

	err = os.MkdirAll(cloneDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating directory %s: %w", cloneDir, err)
	}

	defer func() {
		err = os.RemoveAll(cloneDir)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("Unable to remove working directory")
		}
	}()

	filePath := ".devcontainer/devcontainer.json"
	var catFileOutputBytes []byte
	switch gitspaceConfig.CodeRepoType { //nolint:exhaustive
	case enum.CodeRepoTypeGitness:
		repo, err := s.repoStore.FindByRef(ctx, gitspaceConfig.CodeRepoURL)

		// Backfill clone URL
		repo.GitURL = s.urlProvider.GenerateContainerGITCloneURL(ctx, repo.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to find repository: %w", err)
		}
		resolvedDetails.CloneURL = repo.GitURL
		catFileOutputBytes, err = s.getDevContainerConfigInternal(ctx, gitspaceConfig, filePath, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to read devcontainer file : %w", err)
		}
		netrc, err := s.gitnessCredentials(ctx, repo, gitspaceConfig.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repo credentials: %w", err)
		}
		resolvedDetails.Credentials = netrc
	default:
		catFileOutputBytes, err = s.getDevContainerConfigPublic(ctx, gitspaceConfig, cloneDir, filePath)
		if err != nil {
			return nil, err
		}
	}
	if len(catFileOutputBytes) == 0 {
		resolvedDetails.DevcontainerConfig = &types.DevcontainerConfig{}
		return resolvedDetails, nil
	}
	sanitizedJSON := removeComments(catFileOutputBytes)
	var config *types.DevcontainerConfig
	err = json.Unmarshal(sanitizedJSON, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse devcontainer json: %w", err)
	}
	resolvedDetails.DevcontainerConfig = config
	return resolvedDetails, nil
}

func (s genericSCM) getDevContainerConfigInternal(ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	filePath string,
	repo *types.Repository,
) ([]byte, error) {
	// create read params once
	readParams := git.CreateReadParams(repo)
	treeNodeOutput, err := s.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              gitspaceConfig.Branch,
		Path:                filePath,
		IncludeLatestCommit: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read tree node: %w", err)
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

func (s genericSCM) getDevContainerConfigPublic(ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	cloneDir string,
	filePath string,
) ([]byte, error) {
	log.Info().Msg("Cloning the repository...")
	cmd := command.New("clone",
		command.WithFlag("--branch", gitspaceConfig.Branch),
		command.WithFlag("--no-checkout"),
		command.WithFlag("--depth", "1"),
		command.WithArg(gitspaceConfig.CodeRepoURL),
		command.WithArg(cloneDir),
	)
	if err := cmd.Run(ctx, command.WithDir(cloneDir)); err != nil {
		return nil, fmt.Errorf("failed to clone repository %s: %w", gitspaceConfig.CodeRepoURL, err)
	}

	var lsTreeOutput bytes.Buffer
	lsTreeCmd := command.New("ls-tree",
		command.WithArg("HEAD"),
		command.WithArg(filePath),
	)

	if err := lsTreeCmd.Run(ctx, command.WithDir(cloneDir), command.WithStdout(&lsTreeOutput)); err != nil {
		return nil, fmt.Errorf("failed to list files in repository %s: %w", cloneDir, err)
	}

	if lsTreeOutput.Len() == 0 {
		log.Info().Msg("File not found, returning empty devcontainerConfig")
		return nil, nil
	}

	fields := strings.Fields(lsTreeOutput.String())
	blobSHA := fields[2]

	var catFileOutput bytes.Buffer
	catFileCmd := command.New("cat-file", command.WithFlag("-p"), command.WithArg(blobSHA))
	err := catFileCmd.Run(
		ctx,
		command.WithDir(cloneDir),
		command.WithStderr(io.Discard),
		command.WithStdout(&catFileOutput),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read devcontainer file from path %s: %w", filePath, err)
	}
	return catFileOutput.Bytes(), nil
}

func removeComments(input []byte) []byte {
	blockCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	input = blockCommentRegex.ReplaceAll(input, nil)
	lineCommentRegex := regexp.MustCompile(`//.*`)
	return lineCommentRegex.ReplaceAll(input, nil)
}

func detectDefaultGitBranch(ctx context.Context, gitRepoDir string) (string, error) {
	cmd := command.New("ls-remote",
		command.WithFlag("--symref"),
		command.WithFlag("-q"),
		command.WithArg(gitRepoDir),
		command.WithArg("HEAD"),
	)
	output := &bytes.Buffer{}
	if err := cmd.Run(ctx, command.WithStdout(output)); err != nil {
		return "", fmt.Errorf("failed to ls remote repo")
	}
	var lsRemoteHeadRegexp = regexp.MustCompile(`ref: refs/heads/([^\s]+)\s+HEAD`)
	match := lsRemoteHeadRegexp.FindStringSubmatch(strings.TrimSpace(output.String()))
	if match == nil {
		return "", ErrNoDefaultBranch
	}
	return match[1], nil
}

func validateURL(request CodeRepositoryRequest) error {
	if _, err := url.ParseRequestURI(request.URL); err != nil {
		return err
	}
	return nil
}

func findUserFromUID(ctx context.Context,
	principalStore store.PrincipalStore, userUID string,
) (*types.User, error) {
	return principalStore.FindUserByUID(ctx, userUID)
}

func (s genericSCM) gitnessCredentials(
	ctx context.Context,
	repo *types.Repository,
	userUID string,
) (*Credentials, error) {
	gitspacePrincipal := bootstrap.NewGitspaceServiceSession().Principal
	user, err := findUserFromUID(ctx, s.principalStore, userUID)
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

	cloneURL, err := url.Parse(repo.GitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clone url '%s': %w", cloneURL, err)
	}

	return &Credentials{
		Password: jwtToken,
		Email:    user.Email,
		Name:     user.DisplayName,
	}, nil
}
