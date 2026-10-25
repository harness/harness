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

// Package prcheck reports the shield/security status check of pull requests: it blocks pull
// requests that introduce dependency vulnerabilities at or above the risk threshold when the
// check is required by a branch protection rule.
package prcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/shield"
	"github.com/harness/gitness/events"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// Identifier is the identifier of the status check, to be used in protection rules.
	Identifier = "shield/security"
	// topFindings is the number of findings listed in a failing summary.
	topFindings = 5
	// maxSummary is the maximum summary length accepted by the check store consumers.
	maxSummary = 2048
	// handlerIdleTimeout is generous: the scan runs inside the event handler.
	handlerIdleTimeout = 10 * time.Minute
)

// Differ is the subset of the Shield service used by the check.
type Differ interface {
	Enabled() bool
	PatchThreshold() int
	ScanDiff(ctx context.Context, repo *types.RepositoryCore, baseSHA, headSHA string) (*shield.DiffResult, error)
}

// PullReqFinder finds pull requests.
type PullReqFinder interface {
	Find(ctx context.Context, id int64) (*types.PullReq, error)
}

// RepoFinder finds repositories by id.
type RepoFinder interface {
	FindByID(ctx context.Context, id int64) (*types.RepositoryCore, error)
}

// CheckStore is the subset of the check store used by the check.
type CheckStore interface {
	FindByIdentifier(ctx context.Context, repoID int64, commitSHA string, identifier string) (types.Check, error)
	Upsert(ctx context.Context, check *types.Check) error
}

// CheckReporter announces reported checks.
type CheckReporter interface {
	Reported(ctx context.Context, payload *checkevents.ReportedPayload)
}

// Service reports the shield/security check.
type Service struct {
	shield      Differ
	pullreqs    PullReqFinder
	repos       RepoFinder
	checks      CheckStore
	reporter    CheckReporter
	now         func() int64
	principalID func() int64
}

// New creates the check service.
func New(
	sh Differ,
	pullreqs PullReqFinder,
	repos RepoFinder,
	checks CheckStore,
	reporter CheckReporter,
) *Service {
	return &Service{
		shield:   sh,
		pullreqs: pullreqs,
		repos:    repos,
		checks:   checks,
		reporter: reporter,
		now:      func() int64 { return time.Now().UnixMilli() },
		principalID: func() int64 {
			return bootstrap.NewSystemServiceSession().Principal.ID
		},
	}
}

// RegisterEventListeners runs the check whenever a pull request is created or its source branch moves.
func RegisterEventListeners(
	ctx context.Context,
	instanceID string,
	enabled bool,
	readerFactory *events.ReaderFactory[*pullreqevents.Reader],
	svc *Service,
) error {
	if !enabled {
		return nil
	}
	const group = "gitness:shield:pullreq"
	_, err := readerFactory.Launch(ctx, group, instanceID,
		func(r *pullreqevents.Reader) error {
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(handlerIdleTimeout),
					stream.WithMaxRetries(1),
				))
			_ = r.RegisterCreated(svc.handleCreated)
			_ = r.RegisterBranchUpdated(svc.handleBranchUpdated)
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to launch pull request event reader for shield: %w", err)
	}
	return nil
}

func (s *Service) handleCreated(
	ctx context.Context,
	e *events.Event[*pullreqevents.CreatedPayload],
) error {
	return s.Check(ctx, e.Payload.PullReqID, e.Payload.SourceSHA)
}

func (s *Service) handleBranchUpdated(
	ctx context.Context,
	e *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	return s.Check(ctx, e.Payload.PullReqID, e.Payload.NewSHA)
}

// Check evaluates the pull request at the source commit and reports the status check. It is
// idempotent per head commit: a commit whose check already has a final result is not scanned again.
// Scan failures are reported as an error status and do not fail the event.
func (s *Service) Check(ctx context.Context, pullReqID int64, headSHA string) error {
	if !s.shield.Enabled() || headSHA == "" {
		return nil
	}
	pr, err := s.pullreqs.Find(ctx, pullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request %d: %w", pullReqID, err)
	}
	// forks live in another repository whose objects the scan cannot read through the target repo;
	// a stale event (source moved on) is handled by the event of the newer commit.
	if pr.SourceRepoID == nil || *pr.SourceRepoID != pr.TargetRepoID || pr.SourceSHA != headSHA {
		return nil
	}
	repo, err := s.repos.FindByID(ctx, pr.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo %d: %w", pr.TargetRepoID, err)
	}

	existing, err := s.checks.FindByIdentifier(ctx, repo.ID, headSHA, Identifier)
	switch {
	case err == nil:
		if existing.Status == enum.CheckStatusSuccess || existing.Status == enum.CheckStatusFailure {
			return nil
		}
	case !errors.Is(err, gitness_store.ErrResourceNotFound):
		return fmt.Errorf("failed to look up check: %w", err)
	}

	started := s.now()
	if err = s.report(ctx, repo.ID, headSHA, enum.CheckStatusPending, "shield scan running", started, 0); err != nil {
		return err
	}

	var res *shield.DiffResult
	baseSHA := pr.MergeBaseSHA
	if baseSHA == "" {
		err = errors.New("pull request has no merge base")
	} else {
		res, err = s.shield.ScanDiff(ctx, repo, baseSHA, headSHA)
	}
	status, summary := Decide(res, err, s.shield.PatchThreshold())
	if rerr := s.report(ctx, repo.ID, headSHA, status, summary, started, s.now()); rerr != nil {
		return rerr
	}
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("pullreq_id", pullReqID).Msg("shield pull request scan failed")
	}
	return nil
}

// Decide maps the diff outcome to the check status and summary.
func Decide(res *shield.DiffResult, scanErr error, threshold int) (enum.CheckStatus, string) {
	if scanErr != nil || res == nil {
		return enum.CheckStatusError, truncate("shield scan failed: " + errText(scanErr))
	}
	var blocking []*types.ShieldFinding
	for _, f := range res.New {
		if f.RiskScore >= threshold {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) == 0 {
		return enum.CheckStatusSuccess, fmt.Sprintf("no new vulnerabilities above %d", threshold)
	}
	shield.SortByRisk(blocking)
	var b strings.Builder
	fmt.Fprintf(&b, "%d new vulnerabilities at or above risk %d", len(blocking), threshold)
	for i, f := range blocking {
		if i == topFindings {
			fmt.Fprintf(&b, "\n... and %d more", len(blocking)-topFindings)
			break
		}
		fix := f.MinFixedVersion
		if fix == "" {
			fix = "none"
		}
		fmt.Fprintf(&b, "\n- %s %s: %s (risk %d, fix %s)", f.Package, f.InstalledVersion, f.VulnID, f.RiskScore, fix)
	}
	return enum.CheckStatusFailure, truncate(b.String())
}

func errText(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

func truncate(s string) string {
	if len(s) <= maxSummary {
		return s
	}
	return s[:maxSummary-3] + "..."
}

func (s *Service) report(
	ctx context.Context,
	repoID int64,
	sha string,
	status enum.CheckStatus,
	summary string,
	started, ended int64,
) error {
	now := s.now()
	check := &types.Check{
		CreatedBy:  s.principalID(),
		Created:    now,
		Updated:    now,
		RepoID:     repoID,
		CommitSHA:  sha,
		Identifier: Identifier,
		Status:     status,
		Summary:    summary,
		Started:    started,
		Ended:      ended,
		Metadata:   json.RawMessage(`{}`),
		Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty, Data: json.RawMessage(`{}`)},
	}
	if err := s.checks.Upsert(ctx, check); err != nil {
		return fmt.Errorf("failed to report %s check: %w", Identifier, err)
	}
	s.reporter.Reported(ctx, &checkevents.ReportedPayload{
		Base:       checkevents.Base{RepoID: repoID, SHA: sha},
		Identifier: Identifier,
		Status:     status,
	})
	return nil
}
