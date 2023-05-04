package cron

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/harness/gitness/gitrpc/server"
	"github.com/rs/zerolog/log"
)

// cleanup repository graveyard
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

func AddAllGitRPCCronJobs(cm *CronManager, gitrpcconfig server.Config) error {
	// periodic repository graveyard cleanup
	graveyardpath := filepath.Join(gitrpcconfig.GitRoot, server.ReposGraveyardSubdirName)
	err := cm.NewCronTask(Nightly, func(ctx context.Context) error { return cleanupRepoGraveyard(ctx, graveyardpath) })
	if err != nil {
		return err
	}
	return nil
}
