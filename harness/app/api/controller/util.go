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
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

// createRPCWriteParams creates base write parameters for git write operations.
// TODO: this function should be in git package and should accept params as interface (contract).
func createRPCWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
	disabled bool,
	isInternal bool,
) (git.WriteParams, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetInternalAPIURL(ctx),
		repo.ID,
		session.Principal.ID,
		disabled,
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
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParams(ctx, urlProvider, session, repo, false, false)
}

// CreateRPCInternalWriteParams creates base write parameters for git internal write operations.
// Internal write operations are git pushes that originate from the Harness server.
func CreateRPCInternalWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParams(ctx, urlProvider, session, repo, false, true)
}

// CreateRPCSystemReferencesWriteParams creates base write parameters for write operations
// on system references (e.g. pullreq references).
func CreateRPCSystemReferencesWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParams(ctx, urlProvider, session, repo, true, true)
}

func MapBranch(b git.Branch) (types.Branch, error) {
	return types.Branch{
		Name:   b.Name,
		SHA:    b.SHA,
		Commit: MapCommit(b.Commit),
	}, nil
}

func MapCommit(c *git.Commit) *types.Commit {
	if c == nil {
		return nil
	}

	return &types.Commit{
		SHA:        c.SHA,
		TreeSHA:    c.TreeSHA,
		ParentSHAs: c.ParentSHAs,
		Title:      c.Title,
		Message:    c.Message,
		Author:     MapSignature(c.Author),
		Committer:  MapSignature(c.Committer),
		SignedData: (*types.SignedData)(c.SignedData),
		Stats:      mapStats(c),
	}
}

func MapCommitTag(t git.CommitTag) types.CommitTag {
	var tagger *types.Signature
	if t.Tagger != nil {
		tagger = &types.Signature{}
		*tagger = MapSignature(*t.Tagger)
	}

	return types.CommitTag{
		Name:        t.Name,
		SHA:         t.SHA,
		IsAnnotated: t.IsAnnotated,
		Title:       t.Title,
		Message:     t.Message,
		Tagger:      tagger,
		SignedData:  (*types.SignedData)(t.SignedData),
		Commit:      MapCommit(t.Commit),
	}
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

func MapSignature(s git.Signature) types.Signature {
	return types.Signature{
		Identity: types.Identity(s.Identity),
		When:     s.When,
	}
}

func IdentityFromPrincipalInfo(p types.PrincipalInfo) *git.Identity {
	return &git.Identity{
		Name:  p.DisplayName,
		Email: p.Email,
	}
}

func SystemServicePrincipalInfo() *git.Identity {
	return IdentityFromPrincipalInfo(*bootstrap.NewSystemServiceSession().Principal.ToPrincipalInfo())
}
