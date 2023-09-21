// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cron

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/harness/gitness/gitrpc/server"

	"github.com/rs/zerolog/log"
)

// cleanupRepoGraveyard cleanups repository graveyard.
func cleanupRepoGraveyard(ctx context.Context, graveyardpath string) error {
	logger := log.Ctx(ctx)
	repolist, err := os.ReadDir(graveyardpath)
	if err != nil {
		logger.Warn().Err(err).Msgf("failed to read repos graveyard directory %s", graveyardpath)
		return err
	}
	for _, repo := range repolist {
		// exit early if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := os.RemoveAll(path.Join(graveyardpath, repo.Name())); err != nil {
			logger.Error().Err(err).Msgf("failed to remove repository %s from graveyard", repo.Name())
		} else {
			logger.Info().Msgf("repository %s removed from graveyard", repo.Name())
		}
	}
	return nil
}

func AddAllGitRPCCronJobs(cm *Manager, gitrpcconfig server.Config) error {
	// periodic repository graveyard cleanup
	graveyardpath := filepath.Join(gitrpcconfig.GitRoot, server.ReposGraveyardSubdirName)
	err := cm.NewCronTask(Nightly, func(ctx context.Context) error { return cleanupRepoGraveyard(ctx, graveyardpath) })
	if err != nil {
		return err
	}
	return nil
}
