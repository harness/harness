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

package controller

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

// createRPCWriteParams creates base write parameters for git write operations.
// TODO: this function should be in git package and should accept params as interface (contract)
func createRPCWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.Repository,
	isInternal bool,
) (git.WriteParams, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetInternalAPIURL(ctx),
		repo.ID,
		session.Principal.ID,
		false,
		isInternal,
	)
	if err != nil {
		return git.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return git.WriteParams{
		Actor: git.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}

// CreateRPCExternalWriteParams creates base write parameters for git external write operations.
// External write operations are direct git pushes.
func CreateRPCExternalWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.Repository,
) (git.WriteParams, error) {
	return createRPCWriteParams(ctx, urlProvider, session, repo, false)
}

// CreateRPCInternalWriteParams creates base write parameters for git internal write operations.
// Internal write operations are git pushes that originate from the Gitness server.
func CreateRPCInternalWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.Repository,
) (git.WriteParams, error) {
	return createRPCWriteParams(ctx, urlProvider, session, repo, true)
}

func MapCommit(c *git.Commit) (*types.Commit, error) {
	if c == nil {
		return nil, fmt.Errorf("commit is nil")
	}

	author, err := MapSignature(&c.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to map author: %w", err)
	}

	committer, err := MapSignature(&c.Committer)
	if err != nil {
		return nil, fmt.Errorf("failed to map committer: %w", err)
	}

	parentSHAs := make([]string, len(c.ParentSHAs))
	for i, sha := range c.ParentSHAs {
		parentSHAs[i] = sha.String()
	}

	return &types.Commit{
			SHA:        c.SHA.String(),
			ParentSHAs: parentSHAs,
			Title:      c.Title,
			Message:    c.Message,
			Author:     *author,
			Committer:  *committer,
			Stats:      mapStats(c),
		},
		nil
}

func mapStats(c *git.Commit) *types.CommitStats {
	if len(c.FileStats) == 0 {
		return nil
	}

	var insertions int64
	var deletions int64
	for _, stat := range c.FileStats {
		insertions += stat.Insertions
		deletions += stat.Deletions
	}

	return &types.CommitStats{
		Total: types.ChangeStats{
			Insertions: insertions,
			Deletions:  deletions,
			Changes:    insertions + deletions,
		},
		Files: mapFileStats(c),
	}
}

func mapFileStats(c *git.Commit) []types.CommitFileStats {
	fileStats := make([]types.CommitFileStats, len(c.FileStats))

	for i, fStat := range c.FileStats {
		fileStats[i] = types.CommitFileStats{
			Path:    fStat.Path,
			OldPath: fStat.OldPath,
			Status:  fStat.Status,
			ChangeStats: types.ChangeStats{
				Insertions: fStat.Insertions,
				Deletions:  fStat.Deletions,
				Changes:    fStat.Insertions + fStat.Deletions,
			},
		}
	}

	return fileStats
}

func MapRenameDetails(c *git.RenameDetails) *types.RenameDetails {
	if c == nil {
		return nil
	}
	return &types.RenameDetails{
		OldPath:         c.OldPath,
		NewPath:         c.NewPath,
		CommitShaBefore: c.CommitShaBefore.String(),
		CommitShaAfter:  c.CommitShaAfter.String(),
	}
}

func MapSignature(s *git.Signature) (*types.Signature, error) {
	if s == nil {
		return nil, fmt.Errorf("signature is nil")
	}

	return &types.Signature{
		Identity: types.Identity{
			Name:  s.Identity.Name,
			Email: s.Identity.Email,
		},
		When: s.When,
	}, nil
}
