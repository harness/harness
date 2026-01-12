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

package languageanalyzer

import (
	"context"
	"fmt"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/langstats"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
)

type LanguageAnalyzer interface {
	AnalyzeLanguages(ctx context.Context, repoID int64) error
}

type Analyzer struct {
	tx            dbtx.Transactor
	repoStore     store.RepoStore
	repoFinder    refcache.RepoFinder
	repoLangStore store.RepoLangStore
	gitService    git.Interface
}

func NewAnalyzer(
	tx dbtx.Transactor,
	repoStore store.RepoStore,
	repoFinder refcache.RepoFinder,
	repoLangStore store.RepoLangStore,
	gitService git.Interface,
) Analyzer {
	return Analyzer{
		tx:            tx,
		repoLangStore: repoLangStore,
		repoStore:     repoStore,
		repoFinder:    repoFinder,
		gitService:    gitService,
	}
}

// AnalyzeLanguages performs a language breakdown for a repo at a given ref.
// It operates directly on a bare repo.
func (a Analyzer) AnalyzeLanguages(
	ctx context.Context,
	repoID int64,
) error {
	// Fetch once outside tx for Git UID etc.
	repo, err := a.repoStore.Find(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo by ID: %w", err)
	}

	langStatsOutput, err := a.gitService.GetRepoLanguageStats(
		ctx, &git.GetRepoLanguageStatsParams{
			ReadParams: git.ReadParams{
				RepoUID: repo.GitUID,
			},
			Branch: repo.DefaultBranch,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get repo language stats: %w", err)
	}

	var (
		mainLang string
		maxBytes int64
		rows     []*types.RepoLangStat
	)

	for lang, stat := range langStatsOutput.Stats {
		if lang == "" {
			continue
		}
		if lang == langstats.Unclassified {
			continue
		}
		rows = append(rows, &types.RepoLangStat{
			Language: lang,
			Bytes:    stat.Bytes,
			Files:    stat.Files,
		})
		if stat.Bytes > maxBytes ||
			(stat.Bytes == maxBytes && lang < mainLang) { // tie-breaker: lexicographically smaller
			maxBytes = stat.Bytes
			mainLang = lang
		}
	}

	err = a.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := a.repoLangStore.DeleteByRepoID(ctx, repoID); err != nil {
			return fmt.Errorf("failed to delete repo languages: %w", err)
		}

		if err := a.repoLangStore.InsertByRepoID(ctx, repoID, rows); err != nil {
			return fmt.Errorf("failed to insert repo languages: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to analyze repo languages: %w", err)
	}

	// run opt-lock repo update outside repo lang tx to avoid delete/reinsert on conflict
	_, err = a.repoStore.UpdateOptLock(ctx, repo, func(repository *types.Repository) error {
		repository.Language = mainLang
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update main repo language: %w", err)
	}

	return nil
}

func (a Analyzer) defaultBranchUpdatedHandler(
	ctx context.Context,
	e *events.Event[*repoevents.DefaultBranchUpdatedPayload],
) error {
	// Default branch changes are infrequent; always re-analyze for simplicity.
	if err := a.AnalyzeLanguages(ctx, e.Payload.RepoID); err != nil {
		return fmt.Errorf("failed to analyze repo languages: %w", err)
	}

	return nil
}

func (a *Analyzer) handleEventBranchUpdated(
	ctx context.Context,
	e *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	repo, err := a.repoFinder.FindByID(ctx, e.Payload.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo by ID: %w", err)
	}

	if e.Payload.Ref != api.BranchPrefix+repo.DefaultBranch {
		// Not the default branch, ignore
		return nil
	}

	if err := a.AnalyzeLanguages(ctx, e.Payload.RepoID); err != nil {
		return fmt.Errorf("failed to analyze repo languages: %w", err)
	}

	return nil
}
