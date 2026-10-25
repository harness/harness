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

// Package shield implements the Shield vulnerability scan service: it reads the dependency manifests
// of a commit, matches them against the local OSV database, scores the findings and turns the
// serious ones into tasks.
package shield

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// DefaultPatchThreshold is the risk score from which a finding produces a task.
	DefaultPatchThreshold = 70
	// maxManifestSize is the largest manifest/lockfile that is parsed.
	maxManifestSize = 32 << 20
	// pageSize is the page size used when walking findings.
	pageSize = 100
)

// ErrNotEnabled is returned when Shield is disabled.
var ErrNotEnabled = errors.New("shield is not enabled")

// GitReader is the subset of the git layer used to read manifests.
type GitReader interface {
	ListPaths(ctx context.Context, params *git.ListPathsParams) (*git.ListPathsOutput, error)
	GetTreeNode(ctx context.Context, params *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error)
	GetBlob(ctx context.Context, params *git.GetBlobParams) (*git.GetBlobOutput, error)
}

// TaskService is the subset of the task service used by Shield.
type TaskService interface {
	Create(ctx context.Context, principalID int64, in task.CreateInput) (*types.Task, bool, error)
	Link(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error
	AddComment(ctx context.Context, principalID, taskID int64, text string) error
}

// VulnData provides the local vulnerability data.
type VulnData interface {
	// LoadDB loads the OSV records of the packages' ecosystems, restricted to the given packages.
	LoadDB(pkgs []osv.Package) (*osv.DB, error)
	// Scorer returns a risk scorer over the latest EPSS and KEV data.
	Scorer() *risk.Scorer
}

// Visibility tells whether a repository is public.
type Visibility interface {
	Get(ctx context.Context, resourceType enum.PublicResourceType, resourcePath string) (bool, error)
}

// RepoFinder finds repositories by id.
type RepoFinder interface {
	FindByID(ctx context.Context, id int64) (*types.RepositoryCore, error)
}

// JobScheduler is the subset of the job scheduler used by Shield.
type JobScheduler interface {
	RunJob(ctx context.Context, def job.Definition) error
	AddRecurring(ctx context.Context, jobUID, jobType, cronDef string, maxDur time.Duration) error
}

// Config is the configuration of the scan service.
type Config struct {
	Enabled        bool
	PatchThreshold int
}

// Service runs Shield scans.
type Service struct {
	config     Config
	scans      store.ShieldScanStore
	findings   store.ShieldFindingStore
	tasks      TaskService
	git        GitReader
	data       VulnData
	visibility Visibility
	repos      RepoFinder
	scheduler  JobScheduler
	registry   *osv.Registry
	baseCache  baseCache

	triager *Triager

	now         func() int64
	principalID func() int64

	hook FindingsHook
}

// SetTriager enables the AI triage of findings after each scan.
func (s *Service) SetTriager(t *Triager) { s.triager = t }

// FindingsHook is called after a scan with the stored findings (tasks already ensured).
type FindingsHook func(ctx context.Context, repo *types.RepositoryCore, findings []*types.ShieldFinding)

// SetFindingsHook sets the hook called after every successful scan. It must be set before scans run.
func (s *Service) SetFindingsHook(h FindingsHook) { s.hook = h }

// NewService creates a scan service.
func NewService(
	config Config,
	scans store.ShieldScanStore,
	findings store.ShieldFindingStore,
	tasks TaskService,
	gitReader GitReader,
	data VulnData,
	visibility Visibility,
	repos RepoFinder,
	scheduler JobScheduler,
) *Service {
	if config.PatchThreshold <= 0 {
		config.PatchThreshold = DefaultPatchThreshold
	}
	return &Service{
		config:     config,
		scans:      scans,
		findings:   findings,
		tasks:      tasks,
		git:        gitReader,
		data:       data,
		visibility: visibility,
		repos:      repos,
		scheduler:  scheduler,
		registry:   osv.NewRegistry(),
		now:        func() int64 { return time.Now().UnixMilli() },
		principalID: func() int64 {
			return bootstrap.NewSystemServiceSession().Principal.ID
		},
	}
}

// Enabled reports whether Shield is enabled.
func (s *Service) Enabled() bool { return s.config.Enabled }

// Scan creates a scan of the repository at the commit and runs it synchronously. The returned scan
// is persisted with its final status; scan failures are recorded on it and are not returned as error.
func (s *Service) Scan(
	ctx context.Context,
	repo *types.RepositoryCore,
	commitSHA string,
	trigger enum.ShieldScanTrigger,
) (*types.ShieldScan, error) {
	scan, err := s.createScan(ctx, repo, commitSHA, trigger)
	if err != nil {
		return nil, err
	}
	return s.run(ctx, repo, scan)
}

func (s *Service) createScan(
	ctx context.Context,
	repo *types.RepositoryCore,
	commitSHA string,
	trigger enum.ShieldScanTrigger,
) (*types.ShieldScan, error) {
	if !s.config.Enabled {
		return nil, ErrNotEnabled
	}
	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA == "" {
		return nil, errors.New("commit sha is required")
	}
	now := s.now()
	scan := &types.ShieldScan{
		RepoID:    repo.ID,
		CommitSHA: commitSHA,
		Ref:       "refs/heads/" + repo.DefaultBranch,
		Trigger:   trigger,
		Scanner:   enum.ShieldScannerOSV,
		Status:    enum.ShieldScanStatusPending,
		Started:   now,
		Created:   now,
	}
	if err := s.scans.Create(ctx, scan); err != nil {
		return nil, fmt.Errorf("failed to create shield scan: %w", err)
	}
	return scan, nil
}

// run executes an existing scan row and records the outcome on it.
func (s *Service) run(
	ctx context.Context,
	repo *types.RepositoryCore,
	scan *types.ShieldScan,
) (*types.ShieldScan, error) {
	scan.Status = enum.ShieldScanStatusRunning
	if err := s.scans.Update(ctx, scan); err != nil {
		return nil, fmt.Errorf("failed to mark shield scan running: %w", err)
	}

	packages, findings, err := s.execute(ctx, repo, scan)

	finished := s.now()
	scan.Finished = &finished
	scan.PackagesCount = packages
	scan.FindingsCount = findings
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("repo_id", repo.ID).Int64("scan_id", scan.ID).Msg("shield scan failed")
		scan.Status = enum.ShieldScanStatusFailed
		scan.Error = err.Error()
	} else {
		scan.Status = enum.ShieldScanStatusDone
		scan.Error = ""
		if s.triager != nil {
			// triage refines the results; its failure must not fail the scan.
			if terr := s.triager.Triage(ctx, repo, scan); terr != nil {
				log.Ctx(ctx).Warn().Err(terr).Int64("scan_id", scan.ID).Msg("shield triage failed")
			}
		}
	}
	if uerr := s.scans.Update(ctx, scan); uerr != nil {
		return nil, fmt.Errorf("failed to update shield scan: %w", uerr)
	}
	return scan, nil
}

func (s *Service) execute(
	ctx context.Context,
	repo *types.RepositoryCore,
	scan *types.ShieldScan,
) (packagesCount, findingsCount int, err error) {
	pkgs, readComplete, err := s.readPackages(ctx, repo, scan.CommitSHA)
	if err != nil {
		return 0, 0, err
	}

	exposure, err := s.exposure(ctx, repo)
	if err != nil {
		return len(pkgs), 0, err
	}

	var osvFindings []osv.Finding
	if len(pkgs) > 0 {
		db, lerr := s.data.LoadDB(pkgs)
		if lerr != nil {
			return len(pkgs), 0, fmt.Errorf("failed to load vulnerability database: %w", lerr)
		}
		osvFindings = osv.Match(db, pkgs).Findings
	}

	scorer := s.data.Scorer()
	now := s.now()
	seen := make(map[string]struct{}, len(osvFindings))
	var stored []*types.ShieldFinding
	for _, of := range osvFindings {
		f := FromOSVFinding(of, repo.ID, scan.ID, now)
		if _, dup := seen[f.Fingerprint]; dup {
			continue
		}
		seen[f.Fingerprint] = struct{}{}

		score, bd := scorer.Score(risk.Input{
			CVSSScore:    of.CVSSScore,
			HasCVSS:      of.HasScore,
			Severity:     of.Severity,
			CVEs:         of.CVEs,
			Reachability: risk.ReachabilityUnknown,
			Exposure:     exposure,
		})
		f.RiskScore = score
		f.EPSS = bd.EPSSScore
		f.InKEV = bd.InKEV

		if err = s.findings.Upsert(ctx, f); err != nil {
			return len(pkgs), len(stored), fmt.Errorf("failed to upsert shield finding: %w", err)
		}
		stored = append(stored, f)
	}

	// findings that vanished: only trustworthy if every manifest was read.
	if readComplete {
		if err = s.markFixed(ctx, repo.ID, scan); err != nil {
			return len(pkgs), len(stored), err
		}
	} else {
		log.Ctx(ctx).Warn().Int64("scan_id", scan.ID).
			Msg("shield scan could not read every manifest, skipping fixed detection")
	}

	for _, f := range stored {
		if terr := s.ensureTask(ctx, repo, f); terr != nil {
			// a task failure must not lose the scan results; the next scan retries it.
			log.Ctx(ctx).Warn().Err(terr).Int64("finding_id", f.ID).Msg("failed to create shield task")
		}
	}

	if s.hook != nil {
		s.hook(ctx, repo, stored)
	}

	return len(pkgs), len(stored), nil
}

func (s *Service) exposure(ctx context.Context, repo *types.RepositoryCore) (risk.Exposure, error) {
	public, err := s.visibility.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to check repository visibility: %w", err)
	}
	if public {
		return risk.ExposurePublic, nil
	}
	return risk.ExposureInternal, nil
}

// readPackages parses every supported manifest of the commit. complete is false if any manifest
// could not be read or parsed.
func (s *Service) readPackages(
	ctx context.Context,
	repo *types.RepositoryCore,
	commitSHA string,
) (pkgs []osv.Package, complete bool, err error) {
	read := git.ReadParams{RepoUID: repo.GitUID}
	paths, err := s.git.ListPaths(ctx, &git.ListPathsParams{ReadParams: read, GitREF: commitSHA})
	if err != nil {
		return nil, false, fmt.Errorf("failed to list paths at %s: %w", commitSHA, err)
	}

	complete = true
	for _, p := range paths.Files {
		if !s.registry.Handles(p) {
			continue
		}
		res, perr := s.parseManifest(ctx, read, commitSHA, p)
		if perr != nil {
			complete = false
			log.Ctx(ctx).Warn().Err(perr).Str("path", p).Msg("shield: failed to read manifest")
			continue
		}
		pkgs = append(pkgs, res.Packages...)
	}
	return pkgs, complete, nil
}

func (s *Service) parseManifest(
	ctx context.Context,
	read git.ReadParams,
	commitSHA, path string,
) (*osv.ParseResult, error) {
	node, err := s.git.GetTreeNode(ctx, &git.GetTreeNodeParams{ReadParams: read, GitREF: commitSHA, Path: path})
	if err != nil {
		return nil, err
	}
	blob, err := s.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: read, SHA: node.Node.SHA, SizeLimit: maxManifestSize,
	})
	if err != nil {
		return nil, err
	}
	defer blob.Content.Close()
	if blob.Size > maxManifestSize {
		return nil, fmt.Errorf("manifest too large: %d bytes", blob.Size)
	}
	data, err := io.ReadAll(blob.Content)
	if err != nil {
		return nil, err
	}
	return s.registry.Parse(path, strings.NewReader(string(data)))
}

// markFixed marks the findings not seen by the scan as fixed and records it on their tasks.
func (s *Service) markFixed(ctx context.Context, repoID int64, scan *types.ShieldScan) error {
	// collect first: after MarkFixedNotSeenIn the candidates are no longer open.
	var withTask []*types.ShieldFinding
	for page := 1; ; page++ {
		list, err := s.findings.List(ctx, repoID, &types.ShieldFindingFilter{
			Page: page, Size: pageSize, States: []enum.ShieldFindingState{enum.ShieldFindingStateOpen},
		})
		if err != nil {
			return fmt.Errorf("failed to list open shield findings: %w", err)
		}
		for _, f := range list {
			if f.LastSeenScanID != scan.ID && f.TaskID != nil {
				withTask = append(withTask, f)
			}
		}
		if len(list) < pageSize {
			break
		}
	}

	if _, err := s.findings.MarkFixedNotSeenIn(ctx, repoID, scan.ID, s.now()); err != nil {
		return fmt.Errorf("failed to mark shield findings fixed: %w", err)
	}

	for _, f := range withTask {
		text := fmt.Sprintf("finding no longer detected in scan %d", scan.ID)
		if err := s.tasks.AddComment(ctx, s.principalID(), *f.TaskID, text); err != nil {
			log.Ctx(ctx).Warn().Err(err).Int64("task_id", *f.TaskID).Msg("failed to comment fixed shield finding")
		}
	}
	return nil
}

// PriorityForRisk maps a risk score to a task priority.
func PriorityForRisk(score int) enum.TaskPriority {
	switch {
	case score >= 90:
		return enum.TaskPriorityP0
	case score >= 70:
		return enum.TaskPriorityP1
	case score >= 40:
		return enum.TaskPriorityP2
	default:
		return enum.TaskPriorityP3
	}
}

// ensureTask creates (or finds, deduplicated by fingerprint) the task of an open finding at or above
// the patch threshold, links it and stores it on the finding.
func (s *Service) ensureTask(ctx context.Context, repo *types.RepositoryCore, f *types.ShieldFinding) error {
	if f.State != enum.ShieldFindingStateOpen || f.RiskScore < s.config.PatchThreshold {
		return nil
	}
	principal := s.principalID()
	t, _, err := s.tasks.Create(ctx, principal, task.CreateInput{
		RepoID:      repo.ID,
		Title:       fmt.Sprintf("%s in %s %s", f.VulnID, f.Package, f.InstalledVersion),
		Description: taskDescription(f),
		Origin:      enum.TaskOriginShield,
		OriginRef:   f.Fingerprint,
		Priority:    PriorityForRisk(f.RiskScore),
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	if err = s.tasks.Link(ctx, principal, t.ID, enum.TaskLinkKindShieldFinding,
		strconv.FormatInt(f.ID, 10)); err != nil {
		return fmt.Errorf("failed to link task: %w", err)
	}
	if f.TaskID != nil && *f.TaskID == t.ID {
		return nil
	}
	if err = s.findings.SetTask(ctx, f.ID, &t.ID, s.now()); err != nil {
		return fmt.Errorf("failed to set finding task: %w", err)
	}
	f.TaskID = &t.ID
	return nil
}

func taskDescription(f *types.ShieldFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s affects %s %s (%s) in `%s`.\n\n", f.VulnID, f.Package, f.InstalledVersion, f.Ecosystem,
		f.ManifestPath)
	if f.Summary != "" {
		b.WriteString(f.Summary + "\n\n")
	}
	fmt.Fprintf(&b, "- Risk score: %d/100\n- CVSS: %.1f\n- EPSS: %.4f\n- In CISA KEV: %t\n",
		f.RiskScore, f.CVSSScore, f.EPSS, f.InKEV)
	if f.MinFixedVersion != "" {
		fmt.Fprintf(&b, "- Minimum fixed version: %s\n", f.MinFixedVersion)
	}
	return b.String()
}
