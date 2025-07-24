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

package git

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"

	"github.com/gotidy/ptr"
)

const (
	// FullRepackCooldownPeriod is the cooldown period that needs to pass since the last full
	// repack before we consider doing another full repack.
	FullRepackCooldownPeriod = 5 * 24 * time.Hour

	// LooseObjectLimit is the limit of loose objects we accept both when doing incremental
	// repacks and when pruning objects.
	LooseObjectLimit = 1024
)

type OptimizeRepoStrategy int

const (
	OptimizeRepoStrategyGC        OptimizeRepoStrategy = 0
	OptimizeRepoStrategyHeuristic OptimizeRepoStrategy = 1
	OptimizeRepoStrategyFull      OptimizeRepoStrategy = 2
)

func (s OptimizeRepoStrategy) Validate() error {
	switch s {
	case OptimizeRepoStrategyGC, OptimizeRepoStrategyHeuristic, OptimizeRepoStrategyFull:
		return nil
	default:
		return fmt.Errorf("invalid optimization strategy: %d", s)
	}
}

type OptimizeRepositoryParams struct {
	ReadParams
	Strategy OptimizeRepoStrategy
	GCArgs   map[string]string // additional arguments for git gc command
}

func (p OptimizeRepositoryParams) Validate() error {
	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if err := p.Strategy.Validate(); err != nil {
		return err
	}

	return nil
}

func parseGCArgs(args map[string]string) api.GCParams {
	var params api.GCParams

	for arg, value := range args {
		argLower := strings.ToLower(arg)
		switch argLower {
		case "aggressive":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				params.Aggressive = boolVal
			}
		case "auto":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				params.Auto = boolVal
			}
		case "cruft":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				params.Cruft = &boolVal
			}
		case "max-cruft-size":
			if intVal, err := strconv.ParseUint(value, 10, 64); err == nil {
				params.MaxCruftSize = intVal
			}
		case "prune":
			// Try parsing as time
			if t, err := time.Parse(api.RFC2822DateFormat, value); err == nil {
				params.Prune = t
			} else if boolVal, err := strconv.ParseBool(value); err == nil {
				params.Prune = &boolVal
			} else {
				params.Prune = value
			}
		case "keep-largest-pack":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				params.KeepLargestPack = boolVal
			}
		}
	}

	return params
}

func (s *Service) OptimizeRepository(
	ctx context.Context,
	params OptimizeRepositoryParams,
) error {
	if err := params.Validate(); err != nil {
		return err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	repoInfo, err := api.LoadRepositoryInfo(repoPath)
	if err != nil {
		return fmt.Errorf("loading repository info: %w", err)
	}

	var optimizationStrategy OptimizationStrategy

	switch params.Strategy {
	case OptimizeRepoStrategyGC:
		err := s.git.GC(ctx, repoPath, parseGCArgs(params.GCArgs))
		if err != nil {
			return fmt.Errorf("GC repository error: %w", err)
		}
		return nil
	case OptimizeRepoStrategyHeuristic:
		optimizationStrategy = NewHeuristicalOptimizationStrategy(repoInfo)
	case OptimizeRepoStrategyFull:
		optimizationStrategy = NewFullOptimizationStrategy(repoInfo)
	default:
		return errors.InvalidArgument("invalid strategy provided")
	}

	repackNeeded, repackParams := optimizationStrategy.ShouldRepackObjects(ctx)
	if repackNeeded {
		err := s.repackObjects(ctx, repoPath, repackParams)
		if err != nil {
			return fmt.Errorf("optimizing (repacking) repository failed: %w", err)
		}
	}

	pruneNeeded, pruneParams := optimizationStrategy.ShouldPruneObjects(ctx)
	if pruneNeeded {
		err := s.git.PruneObjects(ctx, repoPath, api.PruneObjectsParams{
			ExpireBefore: pruneParams.ExpireBefore,
		})
		if err != nil {
			return fmt.Errorf("pruning objects failed: %w", err)
		}
	}

	packRefsNeeded := optimizationStrategy.ShouldRepackReferences(ctx)
	if packRefsNeeded {
		err := s.git.PackRefs(ctx, repoPath, api.PackRefsParams{
			All: true,
		})
		if err != nil {
			return fmt.Errorf("packing references failed: %w", err)
		}
	}

	writeGraphNeeded, p, err := optimizationStrategy.ShouldWriteCommitGraph(ctx)
	if err != nil {
		return err
	}

	if writeGraphNeeded {
		cgp := api.CommitGraphParams{
			Action:       api.CommitGraphActionWrite,
			Reachable:    true,
			ChangedPaths: true,
			SizeMultiple: 4,
			Split:        ptr.Of(api.CommitGraphSplitOptionEmpty),
		}
		if p.ReplaceChain {
			cgp.Split = ptr.Of(api.CommitGraphSplitOptionReplace)
		}
		err := s.git.CommitGraph(ctx, repoPath, cgp)
		if err != nil {
			return fmt.Errorf("writing commit graph failed: %w", err)
		}
	}

	return nil
}

type OptimizationStrategy interface {
	ShouldRepackObjects(context.Context) (bool, RepackParams)
	ShouldWriteCommitGraph(ctx context.Context) (bool, WriteCommitGraphParams, error)
	ShouldPruneObjects(context.Context) (bool, PruneObjectsParams)
	ShouldRepackReferences(context.Context) bool
}

type HeuristicalOptimizationStrategy struct {
	info         api.RepositoryInfo
	expireBefore time.Time
}

func NewHeuristicalOptimizationStrategy(
	info api.RepositoryInfo,
) HeuristicalOptimizationStrategy {
	return HeuristicalOptimizationStrategy{
		info:         info,
		expireBefore: time.Now().Add(api.StaleObjectsGracePeriod),
	}
}

func (s HeuristicalOptimizationStrategy) ShouldRepackObjects(
	context.Context,
) (bool, RepackParams) {
	if s.info.PackFiles.Count == 0 && s.info.LooseObjects.Count == 0 {
		return false, RepackParams{}
	}

	fullRepackParams := RepackParams{
		Strategy:            RepackStrategyFullWithCruft,
		WriteBitmap:         true,
		WriteMultiPackIndex: true,
		CruftExpireBefore:   s.expireBefore,
	}

	nonCruftPackFilesCount := s.info.PackFiles.Count - s.info.PackFiles.CruftCount
	timeSinceLastFullRepack := time.Since(s.info.PackFiles.LastFullRepack)

	if nonCruftPackFilesCount > 1 && timeSinceLastFullRepack > FullRepackCooldownPeriod {
		return true, fullRepackParams
	}

	geometricRepackParams := RepackParams{
		Strategy:            RepackStrategyGeometric,
		WriteBitmap:         true,
		WriteMultiPackIndex: true,
	}

	if !s.info.PackFiles.MultiPackIndex.Exists {
		return true, geometricRepackParams
	}

	allowedLowerLimit := 2.0
	allowedUpperLimit := math.Log(float64(s.info.PackFiles.Size)/1024/1024) / math.Log(1.8)
	actualLimit := math.Max(allowedLowerLimit, allowedUpperLimit)

	untrackedPackfiles := s.info.PackFiles.Count - s.info.PackFiles.MultiPackIndex.PackFileCount

	if untrackedPackfiles > uint64(actualLimit) {
		return true, geometricRepackParams
	}

	incrementalRepackParams := RepackParams{
		Strategy:            RepackStrategyIncrementalWithUnreachable,
		WriteBitmap:         false,
		WriteMultiPackIndex: false,
	}

	if s.info.LooseObjects.Count > LooseObjectLimit {
		return true, incrementalRepackParams
	}

	return false, fullRepackParams
}

type WriteCommitGraphParams struct {
	ReplaceChain bool
}

func (s HeuristicalOptimizationStrategy) ShouldWriteCommitGraph(
	ctx context.Context,
) (bool, WriteCommitGraphParams, error) {
	if s.info.References.LooseReferenceCount == 0 && s.info.References.PackedReferenceSize == 0 {
		return false, WriteCommitGraphParams{}, nil
	}

	if shouldPrune, _ := s.ShouldPruneObjects(ctx); shouldPrune {
		return true, WriteCommitGraphParams{
			ReplaceChain: true,
		}, nil
	}

	if needsRepacking, repackCfg := s.ShouldRepackObjects(ctx); needsRepacking {
		return true, WriteCommitGraphParams{
			ReplaceChain: repackCfg.Strategy == RepackStrategyFullWithCruft && !repackCfg.CruftExpireBefore.IsZero(),
		}, nil
	}

	return false, WriteCommitGraphParams{}, nil
}

type PruneObjectsParams struct {
	ExpireBefore time.Time
}

func (s HeuristicalOptimizationStrategy) ShouldPruneObjects(
	context.Context,
) (bool, PruneObjectsParams) {
	if s.info.LooseObjects.StaleCount < LooseObjectLimit {
		return false, PruneObjectsParams{}
	}

	return true, PruneObjectsParams{
		ExpireBefore: s.expireBefore,
	}
}

func (s HeuristicalOptimizationStrategy) ShouldRepackReferences(
	context.Context,
) bool {
	if s.info.References.LooseReferenceCount == 0 {
		return false
	}

	maxVal := max(16, math.Log(float64(s.info.References.PackedReferenceSize)/100)/math.Log(1.15))
	if uint64(maxVal) > s.info.References.LooseReferenceCount { //nolint:gosimple
		return false
	}

	return true
}

type FullOptimizationStrategy struct {
	info         api.RepositoryInfo
	expireBefore time.Time
}

func NewFullOptimizationStrategy(
	info api.RepositoryInfo,
) FullOptimizationStrategy {
	return FullOptimizationStrategy{
		info:         info,
		expireBefore: time.Now().Add(api.StaleObjectsGracePeriod),
	}
}

func (s FullOptimizationStrategy) ShouldRepackObjects(
	context.Context,
) (bool, RepackParams) {
	return true, RepackParams{
		Strategy:            RepackStrategyFullWithCruft,
		WriteBitmap:         true,
		WriteMultiPackIndex: true,
		CruftExpireBefore:   s.expireBefore,
	}
}

func (s FullOptimizationStrategy) ShouldWriteCommitGraph(
	context.Context,
) (bool, WriteCommitGraphParams, error) {
	return true, WriteCommitGraphParams{
		ReplaceChain: true,
	}, nil
}

func (s FullOptimizationStrategy) ShouldPruneObjects(
	context.Context,
) (bool, PruneObjectsParams) {
	return true, PruneObjectsParams{
		ExpireBefore: s.expireBefore,
	}
}

func (s FullOptimizationStrategy) ShouldRepackReferences(
	context.Context,
) bool {
	return true
}
