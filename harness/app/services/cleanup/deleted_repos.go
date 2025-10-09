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

package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	jobTypeDeletedRepos        = "gitness:cleanup:deleted-repos"
	jobCronDeletedRepos        = "50 0 * * *" // At minute 50 past midnight every day.
	jobMaxDurationDeletedRepos = 1 * time.Hour
	maxDeletedRepoRetrival     = 1000 // to avoid loading all deleted repos in memory at once
)

type deletedReposCleanupJob struct {
	retentionTime time.Duration

	repoStore store.RepoStore
	repoCtrl  *repo.Controller
}

func newDeletedReposCleanupJob(
	retentionTime time.Duration,
	repoStore store.RepoStore,
	repoCtrl *repo.Controller,
) *deletedReposCleanupJob {
	return &deletedReposCleanupJob{
		retentionTime: retentionTime,

		repoStore: repoStore,
		repoCtrl:  repoCtrl,
	}
}

// Handle purges old deleted repositories that are past the retention time.
func (j *deletedReposCleanupJob) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	olderThan := time.Now().Add(-j.retentionTime)

	log.Ctx(ctx).Info().Msgf(
		"start purging deleted repositories older than %s (aka created before %s)",
		j.retentionTime,
		olderThan.Format(time.RFC3339Nano))

	deletedBeforeOrAt := olderThan.UnixMilli()
	session := bootstrap.NewSystemServiceSession()

	purgedRepos := 0
	for {
		filter := &types.RepoFilter{
			Page:              1,
			Size:              maxDeletedRepoRetrival,
			Query:             "",
			Order:             enum.OrderDesc,
			Sort:              enum.RepoAttrDeleted,
			DeletedBeforeOrAt: &deletedBeforeOrAt,
		}
		toBePurgedRepos, err := j.repoStore.ListAll(ctx, filter)
		if err != nil {
			return "", fmt.Errorf("failed to list ready-to-delete repositories: %w", err)
		}

		if len(toBePurgedRepos) == 0 {
			break
		}

		log.Ctx(ctx).Info().Msgf("found %d deleted repositories ready to be purged.", len(toBePurgedRepos))

		for _, r := range toBePurgedRepos {
			deletedBeforeOrAt = *r.Deleted - 1 // to avoid infinite loop if last repo wasn't purged successfully
			err := j.repoCtrl.PurgeNoAuth(ctx, session, r)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf("failed to purge repo with identifier: %s, path: %s, deleted at: %d",
					r.Identifier, r.Path, *r.Deleted)
				continue
			}
			log.Ctx(ctx).Info().Msgf("successfully purged repo with identifier: %s, path: %s, deleted at: %d",
				r.Identifier, r.Path, *r.Deleted)
			purgedRepos++
		}
	}

	result := "no old deleted repositories found"
	if purgedRepos > 0 {
		result = fmt.Sprintf("purged %d deleted repositories", purgedRepos)
	}

	log.Ctx(ctx).Info().Msg(result)

	return result, nil
}
