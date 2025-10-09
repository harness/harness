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
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/harness/gitness/git/api"

	"github.com/rs/zerolog/log"
)

type RepackStrategy string

const (
	RepackStrategyIncrementalWithUnreachable RepackStrategy = "incremental_with_unreachable"
	RepackStrategyFullWithCruft              RepackStrategy = "full_with_cruft"
	RepackStrategyFullWithUnreachable        RepackStrategy = "full_with_unreachable"
	RepackStrategyGeometric                  RepackStrategy = "geometric"
)

type RepackParams struct {
	Strategy            RepackStrategy
	WriteBitmap         bool
	WriteMultiPackIndex bool
	CruftExpireBefore   time.Time
}

func (p RepackParams) Validate() error {
	switch p.Strategy {
	case RepackStrategyIncrementalWithUnreachable:
		if p.WriteBitmap {
			return errors.New("cannot create bitmap with incremental repack strategy")
		}
		if p.WriteMultiPackIndex {
			return errors.New("cannot create multi-pack index with incremental repack strategy")
		}
	case RepackStrategyFullWithCruft:
		if p.CruftExpireBefore.IsZero() {
			return errors.New("cannot repack with cruft with empty expiration time")
		}
	case RepackStrategyFullWithUnreachable:
	case RepackStrategyGeometric:
	default:
		return fmt.Errorf("unknown strategy: %q", p.Strategy)
	}
	return nil
}

func (s *Service) repackObjects(
	ctx context.Context,
	repoPath string,
	params RepackParams,
) error {
	if err := params.Validate(); err != nil {
		return err
	}

	defer func() {
		err := api.SetLastFullRepackTime(repoPath, time.Now())
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to set last full repack time: %s", err.Error())
		}
	}()

	switch params.Strategy {
	case RepackStrategyIncrementalWithUnreachable:
		// Pack all loose objects into a new pack file, regardless of their reachability.
		err := s.git.PackObjects(ctx, repoPath, api.PackObjectsParams{
			PackLooseUnreachable: true,
			Local:                true,
			Incremental:          true,
			NonEmpty:             true,
			Quiet:                true,
		}, filepath.Join(repoPath, "objects", "pack", "pack"))
		if err != nil {
			return fmt.Errorf("incremental with unreachable repack objects error: %w", err)
		}

		// ensure that packed loose objects are deleted.
		err = s.git.PrunePacked(ctx, repoPath, api.PrunePackedParams{
			Quiet: true,
		})
		if err != nil {
			return fmt.Errorf("prune pack objects error: %w", err)
		}
	case RepackStrategyFullWithCruft:
		repackParams := api.RepackParams{
			Cruft:                  true,
			PackKeptObjects:        true,
			Local:                  true,
			RemoveRedundantObjects: true,
			WriteMidx:              params.WriteMultiPackIndex,
		}
		if !params.CruftExpireBefore.IsZero() {
			repackParams.CruftExpireBefore = params.CruftExpireBefore
		}
		err := s.git.RepackObjects(ctx, repoPath, repackParams)
		if err != nil {
			return fmt.Errorf("full with cruft repack objects error: %w", err)
		}
	case RepackStrategyFullWithUnreachable:
		err := s.git.RepackObjects(ctx, repoPath, api.RepackParams{
			SinglePack:             true,
			Local:                  true,
			RemoveRedundantObjects: true,
			KeepUnreachable:        true,
			WriteMidx:              params.WriteMultiPackIndex,
		})
		if err != nil {
			return fmt.Errorf("full with unreachable repack objects error: %w", err)
		}
	case RepackStrategyGeometric:
		err := s.git.RepackObjects(ctx, repoPath, api.RepackParams{
			Geometric:              2,
			RemoveRedundantObjects: true,
			Local:                  true,
			WriteMidx:              params.WriteMultiPackIndex,
		})
		if err != nil {
			return fmt.Errorf("geometric repack objects error: %w", err)
		}
	}

	return nil
}
