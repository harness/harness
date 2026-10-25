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

package shield

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	scanJobTimeout = 30 * time.Minute
	scanJobRetries = 1
	// branchPrefix is the prefix of branch references.
	branchPrefix = "refs/heads/"
)

type scanJobInput struct {
	RepoID int64 `json:"repo_id"`
	ScanID int64 `json:"scan_id"`
}

// Enqueue creates a pending scan of the repository at the commit and schedules a job that runs it.
// The scan row is returned immediately.
func (s *Service) Enqueue(
	ctx context.Context,
	repo *types.RepositoryCore,
	commitSHA string,
	trigger enum.ShieldScanTrigger,
) (*types.ShieldScan, error) {
	scan, err := s.createScan(ctx, repo, commitSHA, trigger)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(scanJobInput{RepoID: repo.ID, ScanID: scan.ID})
	if err != nil {
		return nil, err
	}
	// unique per repo + commit + scan: a re-delivered event creates a new scan row but
	// never collides with a job that is still around.
	def := job.Definition{
		UID:        fmt.Sprintf("shield-scan-%d-%s-%d", repo.ID, scan.CommitSHA, scan.ID),
		Type:       JobTypeScan,
		MaxRetries: scanJobRetries,
		Timeout:    scanJobTimeout,
		Data:       string(data),
	}
	if err = s.scheduler.RunJob(ctx, def); err != nil {
		scan.Status = enum.ShieldScanStatusFailed
		scan.Error = "failed to schedule scan: " + err.Error()
		now := s.now()
		scan.Finished = &now
		if uerr := s.scans.Update(ctx, scan); uerr != nil {
			log.Ctx(ctx).Warn().Err(uerr).Msg("failed to mark unscheduled shield scan failed")
		}
		return nil, fmt.Errorf("failed to schedule shield scan job: %w", err)
	}
	return scan, nil
}

// scanHandler is the job handler of scan jobs.
type scanHandler struct{ s *Service }

// Handle implements job.Handler.
func (h scanHandler) Handle(ctx context.Context, input string, _ job.ProgressReporter) (string, error) {
	var in scanJobInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid shield scan job input: %w", err)
	}
	repo, err := h.s.repos.FindByID(ctx, in.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo %d: %w", in.RepoID, err)
	}
	scan, err := h.s.scans.Find(ctx, in.ScanID)
	if err != nil {
		return "", fmt.Errorf("failed to find scan %d: %w", in.ScanID, err)
	}
	scan, err = h.s.run(ctx, repo, scan)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("scan %d: %s, %d findings", scan.ID, scan.Status, scan.FindingsCount), nil
}

// Register registers the job handlers and, if enabled, the recurring data sync.
func Register(
	ctx context.Context,
	enabled bool,
	interval time.Duration,
	scheduler JobScheduler,
	executor *job.Executor,
	syncer *Syncer,
	svc *Service,
) error {
	if err := executor.Register(JobTypeScan, scanHandler{s: svc}); err != nil {
		return fmt.Errorf("failed to register shield scan job: %w", err)
	}
	if err := executor.Register(JobTypeSync, syncer); err != nil {
		return fmt.Errorf("failed to register shield sync job: %w", err)
	}
	if !enabled {
		return nil
	}
	if err := scheduler.AddRecurring(ctx, JobTypeSync, JobTypeSync, syncCron(interval), syncJobTimeout); err != nil {
		return fmt.Errorf("failed to schedule shield sync job: %w", err)
	}
	// populate the data right away instead of waiting for the first cron tick.
	def := job.Definition{
		UID:     fmt.Sprintf("%s-startup-%d", JobTypeSync, time.Now().Unix()),
		Type:    JobTypeSync,
		Timeout: syncJobTimeout,
	}
	if err := scheduler.RunJob(ctx, def); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to schedule initial shield data sync")
	}
	return nil
}

// syncCron turns the interval into a cron expression (whole hours, at least 1h).
func syncCron(interval time.Duration) string {
	hours := int((interval + time.Hour - 1) / time.Hour)
	switch {
	case hours <= 1:
		return "0 * * * *"
	case hours >= 24:
		return "0 0 * * *"
	default:
		return fmt.Sprintf("0 */%d * * *", hours)
	}
}

// RegisterEventListeners scans the default branch whenever it receives a push.
func RegisterEventListeners(
	ctx context.Context,
	instanceID string,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	svc *Service,
) error {
	const group = "gitness:shield:git"
	_, err := gitReaderFactory.Launch(ctx, group, instanceID,
		func(r *gitevents.Reader) error {
			const idleTimeout = 15 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterBranchCreated(svc.handleBranchCreated)
			_ = r.RegisterBranchUpdated(svc.handleBranchUpdated)
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to launch git event reader for shield: %w", err)
	}
	return nil
}

func (s *Service) handleBranchCreated(
	ctx context.Context,
	e *events.Event[*gitevents.BranchCreatedPayload],
) error {
	return s.onPush(ctx, e.Payload.RepoID, e.Payload.Ref, e.Payload.SHA)
}

func (s *Service) handleBranchUpdated(
	ctx context.Context,
	e *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	return s.onPush(ctx, e.Payload.RepoID, e.Payload.Ref, e.Payload.NewSHA)
}

// onPush schedules a scan when the default branch of the repository moved to sha.
func (s *Service) onPush(ctx context.Context, repoID int64, ref, sha string) error {
	if !s.config.Enabled || sha == "" {
		return nil
	}
	repo, err := s.repos.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo %d: %w", repoID, err)
	}
	if ref != branchPrefix+repo.DefaultBranch {
		return nil
	}
	// a replayed event must not start a second scan of the same commit.
	recent, err := s.scans.ListByRepo(ctx, repoID, 1, 10)
	if err != nil {
		return fmt.Errorf("failed to list shield scans: %w", err)
	}
	for _, sc := range recent {
		if sc.CommitSHA == sha && sc.Trigger == enum.ShieldScanTriggerPush &&
			sc.Status != enum.ShieldScanStatusFailed {
			return nil
		}
	}
	if _, err = s.Enqueue(ctx, repo, sha, enum.ShieldScanTriggerPush); err != nil {
		return err
	}
	return nil
}
