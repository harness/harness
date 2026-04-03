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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CreateRPCGitPushWriteParams creates base write parameters for git push operations from git clients.
func CreateRPCGitPushWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, false, enum.GitOpTypeGitPush,
	)
}

// CreateRPCAPIContentWriteParams creates base write parameters for API content operations:
// commit API calls and apply comment suggestions.
func CreateRPCAPIContentWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, false, enum.GitOpTypeAPIContent,
	)
}

// CreateRPCAPIContentBypassRulesWriteParams creates base write parameters for API content
// operations with rule bypass. Used for commit API calls with push rules bypass.
func CreateRPCAPIContentBypassRulesWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, false, enum.GitOpTypeAPIContentBypassRules,
	)
}

// CreateRPCAPIRefsWriteParams creates base write parameters for API reference operations:
// branch/tag management and PR merge operations.
func CreateRPCAPIRefsWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, false, enum.GitOpTypeAPIRefsOnly,
	)
}

// CreateRPCSystemReferencesWriteParams creates base write parameters for write operations
// on system references (e.g. pullreq references).
func CreateRPCSystemReferencesWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	// System references are disabled from hooks
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, true, enum.GitOpTypeAPISystemRefs,
	)
}

// CreateRPCAPILinkedSyncWriteParams creates base write parameters for linked repository
// synchronization operations.
func CreateRPCAPILinkedSyncWriteParams(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
) (git.WriteParams, error) {
	return createRPCWriteParamsWithOperationType(
		ctx, urlProvider, session, repo, false, enum.GitOpTypeAPILinkedSync,
	)
}

// createRPCWriteParamsWithOperationType creates base write parameters for git write operations.
func createRPCWriteParamsWithOperationType(
	ctx context.Context,
	urlProvider url.Provider,
	session *auth.Session,
	repo *types.RepositoryCore,
	disabled bool,
	operationType enum.GitOpType,
) (git.WriteParams, error) {
	return githook.CreateWriteParamsForOperation(
		ctx,
		urlProvider.GetInternalAPIURL(ctx),
		git.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		repo.ID,
		repo.GitUID,
		session.Principal.ID,
		disabled,
		operationType,
	)
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
