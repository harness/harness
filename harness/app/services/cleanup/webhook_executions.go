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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"

	"github.com/rs/zerolog/log"
)

const (
	jobTypeWebhookExecutions        = "gitness:cleanup:webhook-executions"
	jobCronWebhookExecutions        = "21 */4 * * *" // At minute 21 past every 4th hour.
	jobMaxDurationWebhookExecutions = 1 * time.Minute
)

type webhookExecutionsCleanupJob struct {
	retentionTime time.Duration

	webhookExecutionStore store.WebhookExecutionStore
}

func newWebhookExecutionsCleanupJob(
	retentionTime time.Duration,
	webhookExecutionStore store.WebhookExecutionStore,
) *webhookExecutionsCleanupJob {
	return &webhookExecutionsCleanupJob{
		retentionTime: retentionTime,

		webhookExecutionStore: webhookExecutionStore,
	}
}

// Handle purges old webhook executions that are past the retention time.
func (j *webhookExecutionsCleanupJob) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	olderThan := time.Now().Add(-j.retentionTime)

	log.Ctx(ctx).Info().Msgf(
		"start purging webhook executions older than %s (aka created before %s)",
		j.retentionTime,
		olderThan.Format(time.RFC3339Nano))

	n, err := j.webhookExecutionStore.DeleteOld(ctx, olderThan)
	if err != nil {
		return "", fmt.Errorf("failed to delete old webhook executions: %w", err)
	}

	result := "no old webhook executions found"
	if n > 0 {
		result = fmt.Sprintf("deleted %d webhook executions", n)
	}

	log.Ctx(ctx).Info().Msg(result)

	return result, nil
}
