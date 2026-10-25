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

package bump

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// JobTypeBump is the job type of bump jobs.
	JobTypeBump = "shield_bump"

	bumpJobTimeout = 10 * time.Minute
	branchPrefix   = "harness-agent/shield/"
	// maxBranchPart bounds the length of the vuln id / package parts of the branch name.
	maxBranchPart = 60
)

// Result is the outcome of one bump run.
type Result struct {
	// Skipped is set (with a reason) if nothing was done.
	Skipped string
	Branch  string
	PRNum   int64
	PRID    int64
}

// FileChange is a file to commit.
type FileChange struct {
	Path    string
	Content []byte
	// BlobSHA is the current blob of the file on the base commit.
	BlobSHA string
}

// File is a file read from the repository.
type File struct {
	Content []byte
	BlobSHA string
}

// Repository is the git access needed by the service. Writes are done as the agent principal.
type Repository interface {
	// HeadSHA returns the head commit of the branch.
	HeadSHA(ctx context.Context, repo *types.RepositoryCore, branch string) (string, error)
	// ReadFile reads a file at the ref; the boolean is false if it does not exist.
	ReadFile(ctx context.Context, repo *types.RepositoryCore, ref, filePath string) (File, bool, error)
	BranchExists(ctx context.Context, repo *types.RepositoryCore, branch string) (bool, error)
	CreateBranch(ctx context.Context, repo *types.RepositoryCore, branch, fromSHA string) error
	// Commit commits the files on the branch as the agent principal.
	Commit(ctx context.Context, repo *types.RepositoryCore, branch, message string, files []FileChange) error
}

// PullRequestInput is the input of PullRequests.Create.
type PullRequestInput struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	IsDraft      bool
}

// PullRequest identifies a created pull request.
type PullRequest struct {
	ID     int64
	Number int64
}

// PullRequests creates pull requests as the agent principal.
type PullRequests interface {
	HasOpen(ctx context.Context, repo *types.RepositoryCore, sourceBranch, targetBranch string) (bool, error)
	Create(ctx context.Context, repo *types.RepositoryCore, in PullRequestInput) (PullRequest, error)
}

// Guard is the subset of agentguard.Guard used by the service.
type Guard interface {
	Check(ctx context.Context, req agentguard.Request) agentguard.Decision
	StartAttempt(ctx context.Context, req agentguard.Request) (agentguard.Attempt, error)
	FinishAttempt(ctx context.Context, attemptID int64, status agentguard.AttemptStatus) error
}

// Tasks is the subset of the task service and store used by the service.
type Tasks interface {
	Find(ctx context.Context, id int64) (*types.Task, error)
	Link(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error
	AddComment(ctx context.Context, principalID, taskID int64, text string) error
}

// Findings is the subset of the finding store used by the service.
type Findings interface {
	Find(ctx context.Context, id int64) (*types.ShieldFinding, error)
}

// Repos finds repositories.
type Repos interface {
	FindByID(ctx context.Context, id int64) (*types.RepositoryCore, error)
}

// Scheduler schedules jobs.
type Scheduler interface {
	RunJob(ctx context.Context, def job.Definition) error
}

// Config configures the service.
type Config struct {
	Enabled bool
	// PatchThreshold is the minimum risk score of findings that are bumped.
	PatchThreshold int
	// GoProxy is the module proxy base URL (default DefaultGoProxy).
	GoProxy string
}

// Service creates dependency bump pull requests for Shield findings.
type Service struct {
	config   Config
	guard    Guard
	settings agentguard.Settings
	repo     Repository
	prs      PullRequests
	tasks    Tasks
	findings Findings
	repos    Repos
	sched    Scheduler
	proxy    *GoProxy

	agent func() *types.Principal
	now   func() time.Time
}

// NewService creates the service. agent returns the harness-agent principal.
func NewService(
	config Config,
	guard Guard,
	agentSettings agentguard.Settings,
	repo Repository,
	prs PullRequests,
	tasks Tasks,
	findings Findings,
	repos Repos,
	sched Scheduler,
	agent func() *types.Principal,
) *Service {
	return &Service{
		config:   config,
		guard:    guard,
		settings: agentSettings,
		repo:     repo,
		prs:      prs,
		tasks:    tasks,
		findings: findings,
		repos:    repos,
		sched:    sched,
		proxy:    NewGoProxy(config.GoProxy),
		agent:    agent,
		now:      time.Now,
	}
}

// SetProxy overrides the module proxy client (tests).
func (s *Service) SetProxy(p *GoProxy) { s.proxy = p }

// Eligible tells whether the finding can be handled by a deterministic bump: open, a bump
// strategy with a fixed version on the same compatibility line, and a task to attach the PR to.
func Eligible(f *types.ShieldFinding, threshold int) bool {
	if f == nil || f.State != enum.ShieldFindingStateOpen || f.TaskID == nil || f.RiskScore < threshold {
		return false
	}
	if f.FixStrategy == nil || *f.FixStrategy != enum.ShieldFixStrategyBump || f.MinFixedVersion == "" {
		return false
	}
	return sameLine(f.Ecosystem, f.InstalledVersion, f.MinFixedVersion)
}

func sameLine(ecosystem, from, to string) bool {
	c := osv.ComparerFor(ecosystem)
	if c == nil {
		return false
	}
	lf, err1 := c.Line(from)
	lt, err2 := c.Line(to)
	if err1 != nil || err2 != nil || lf != lt {
		return false
	}
	cmp, err := c.Compare(from, to)
	return err == nil && cmp < 0
}

// BranchName returns the bump branch of a finding.
func BranchName(f *types.ShieldFinding) string {
	return branchPrefix + sanitizeBranchPart(f.VulnID) + "-" + sanitizeBranchPart(f.Package)
}

var (
	branchBadRe  = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	branchDotsRe = regexp.MustCompile(`\.{2,}`)
)

func sanitizeBranchPart(s string) string {
	s = branchBadRe.ReplaceAllString(s, "-")
	s = branchDotsRe.ReplaceAllString(s, ".")
	if len(s) > maxBranchPart {
		s = s[:maxBranchPart]
	}
	s = strings.Trim(s, "-.")
	s = strings.TrimSuffix(s, ".lock")
	if s == "" {
		s = "x"
	}
	return s
}

func taskKey(findingID int64) string { return fmt.Sprintf("shield-bump:%d", findingID) }

// Bump creates the bump pull request of the finding. It is idempotent: if an open pull request for
// the finding's branch exists nothing is done (Result.Skipped is set).
//
//nolint:gocognit,funlen // linear flow with a refusal per step
func (s *Service) Bump(ctx context.Context, repo *types.RepositoryCore, f *types.ShieldFinding) (*Result, error) {
	if !s.config.Enabled {
		return &Result{Skipped: "shield bump disabled"}, nil
	}
	if !Eligible(f, 0) {
		return &Result{Skipped: "finding is not eligible for a deterministic bump"}, nil
	}
	agent := s.agent()
	if agent == nil {
		return &Result{Skipped: string(agentguard.ReasonAgentNotConfigured)}, nil
	}

	branch := BranchName(f)
	open, err := s.prs.HasOpen(ctx, repo, branch, repo.DefaultBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to check open pull requests: %w", err)
	}
	if open {
		return &Result{Skipped: "pull request already open", Branch: branch}, nil
	}

	req := agentguard.Request{
		RepoID:      repo.ID,
		SpacePath:   spacePath(repo.Path),
		RepoPath:    repo.Path,
		TaskKey:     taskKey(f.ID),
		MinAutonomy: settings.AutonomyDraftPR,
	}
	if d := s.guard.Check(ctx, req); !d.Allowed {
		return &Result{Skipped: string(d.Reason) + ": " + d.Detail}, nil
	}
	agentSettings, err := s.settings.AgentSettings(ctx, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent settings: %w", err)
	}
	draft := *agentSettings.Autonomy == settings.AutonomyDraftPR

	attempt, err := s.guard.StartAttempt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start attempt: %w", err)
	}
	res, err := s.run(ctx, repo, f, agent, branch, draft)
	status := agentguard.AttemptSucceeded
	if err != nil {
		status = agentguard.AttemptFailed
	}
	if ferr := s.guard.FinishAttempt(ctx, attempt.ID, status); ferr != nil {
		log.Ctx(ctx).Warn().Err(ferr).Int64("finding_id", f.ID).Msg("failed to finish shield bump attempt")
	}
	if err != nil && IsNeedsAgent(err) && f.TaskID != nil {
		text := "Deterministic bump not possible, the fix agent is needed: " + err.Error()
		if cerr := s.tasks.AddComment(ctx, agent.ID, *f.TaskID, text); cerr != nil {
			log.Ctx(ctx).Warn().Err(cerr).Int64("finding_id", f.ID).Msg("failed to comment bump refusal")
		}
	}
	return res, err
}

func (s *Service) run(
	ctx context.Context,
	repo *types.RepositoryCore,
	f *types.ShieldFinding,
	agent *types.Principal,
	branch string,
	draft bool,
) (*Result, error) {
	t, err := s.tasks.Find(ctx, *f.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	head, err := s.repo.HeadSHA(ctx, repo, repo.DefaultBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve default branch: %w", err)
	}
	changes, err := s.plan(ctx, repo, head, f)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.BranchExists(ctx, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to check branch: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("branch %q already exists without an open pull request", branch)
	}
	if err = s.repo.CreateBranch(ctx, repo, branch, head); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	title := CommitTitle(f)
	if err = s.repo.Commit(ctx, repo, branch, title, changes); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}
	pr, err := s.prs.Create(ctx, repo, PullRequestInput{
		Title:        title,
		Description:  Description(f, t.Number),
		SourceBranch: branch,
		TargetBranch: repo.DefaultBranch,
		IsDraft:      draft,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// the tasklink service links from the "Fixes TASK-n" text as well; linking here is idempotent
	// and keeps the task consistent if that event is delayed.
	if err = s.tasks.Link(ctx, agent.ID, t.ID, enum.TaskLinkKindPullReq, strconv.FormatInt(pr.ID, 10)); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", t.ID).Msg("failed to link bump pull request")
	}
	text := fmt.Sprintf("Opened pull request #%d (%s) bumping %s to %s without using AI.",
		pr.Number, draftWord(draft), f.Package, f.MinFixedVersion)
	if err = s.tasks.AddComment(ctx, agent.ID, t.ID, text); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", t.ID).Msg("failed to comment bump pull request")
	}
	return &Result{Branch: branch, PRNum: pr.Number, PRID: pr.ID}, nil
}

func draftWord(draft bool) string {
	if draft {
		return "draft"
	}
	return "ready"
}

// CommitTitle is the commit message and pull request title of the bump.
func CommitTitle(f *types.ShieldFinding) string {
	return fmt.Sprintf("fix(deps): bump %s to %s (%s)", f.Package, f.MinFixedVersion, strings.Join(vulnIDs(f), ", "))
}

func vulnIDs(f *types.ShieldFinding) []string {
	ids := []string{f.VulnID}
	seen := map[string]bool{f.VulnID: true}
	for _, a := range f.Aliases {
		if strings.HasPrefix(a, "CVE-") && !seen[a] {
			seen[a] = true
			ids = append(ids, a)
		}
	}
	return ids
}

// Description is the pull request description.
func Description(f *types.ShieldFinding, taskNumber int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Fixes TASK-%d\n\n", taskNumber)
	fmt.Fprintf(&b, "Bumps `%s` from %s to %s (minimum fixed version, same major line) in `%s`.\n\n",
		f.Package, f.InstalledVersion, f.MinFixedVersion, f.ManifestPath)
	if f.Summary != "" {
		b.WriteString(f.Summary + "\n\n")
	}
	fmt.Fprintf(&b, "- Vulnerability: %s\n- Risk score: %d/100\n- CVSS: %.1f\n- In CISA KEV: %t\n",
		strings.Join(vulnIDs(f), ", "), f.RiskScore, f.CVSSScore, f.InKEV)
	b.WriteString("\nThis change was generated deterministically by Shield without using AI. " +
		"Transitive dependencies were not re-resolved: let CI validate the build and tests.\n")
	return b.String()
}

// plan computes the file changes of the bump on the base commit.
func (s *Service) plan(
	ctx context.Context,
	repo *types.RepositoryCore,
	ref string,
	f *types.ShieldFinding,
) ([]FileChange, error) {
	manifest, ok, err := s.repo.ReadFile(ctx, repo, ref, f.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", f.ManifestPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s does not exist", ErrUnsupported, f.ManifestPath)
	}
	base := path.Base(f.ManifestPath)
	switch {
	case f.Ecosystem == osv.EcosystemGo && base == "go.mod":
		sumPath := path.Join(path.Dir(f.ManifestPath), "go.sum")
		sum, _, err := s.repo.ReadFile(ctx, repo, ref, sumPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", sumPath, err)
		}
		res, err := BumpGoMod(ctx, s.proxy, manifest.Content, sum.Content, f.Package, f.MinFixedVersion)
		if err != nil {
			return nil, err
		}
		changes := []FileChange{{Path: f.ManifestPath, Content: res.GoMod, BlobSHA: manifest.BlobSHA}}
		sumChange := FileChange{Path: sumPath, Content: res.GoSum, BlobSHA: sum.BlobSHA}
		return append(changes, sumChange), nil
	case f.Ecosystem == osv.EcosystemPyPI && isRequirementsFile(base):
		out, err := BumpRequirements(manifest.Content, f.Package, f.InstalledVersion, f.MinFixedVersion)
		if err != nil {
			return nil, err
		}
		return []FileChange{{Path: f.ManifestPath, Content: out, BlobSHA: manifest.BlobSHA}}, nil
	case f.Ecosystem == osv.EcosystemNPM:
		return nil, needsAgent("%s: npm lockfiles cannot be updated deterministically", f.ManifestPath)
	default:
		return nil, fmt.Errorf("%w: %s (%s)", ErrUnsupported, f.ManifestPath, f.Ecosystem)
	}
}

func isRequirementsFile(base string) bool {
	return strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt")
}

func spacePath(repoPath string) string {
	if i := strings.LastIndex(repoPath, "/"); i >= 0 {
		return repoPath[:i]
	}
	return ""
}

// ---------------------------------------------------------------------
// trigger and job

type jobInput struct {
	RepoID    int64 `json:"repo_id"`
	FindingID int64 `json:"finding_id"`
}

// OnFindings enqueues one bump job per eligible finding of a finished scan. It is idempotent: the
// job id is unique per finding and day and Bump skips findings with an open pull request.
func (s *Service) OnFindings(ctx context.Context, repo *types.RepositoryCore, findings []*types.ShieldFinding) {
	if !s.config.Enabled {
		return
	}
	for _, f := range findings {
		if !Eligible(f, s.config.PatchThreshold) {
			continue
		}
		data, err := json.Marshal(jobInput{RepoID: repo.ID, FindingID: f.ID})
		if err != nil {
			continue
		}
		def := job.Definition{
			UID: fmt.Sprintf("%s-%d-%d-%d", JobTypeBump, repo.ID, f.ID,
				s.now().Unix()/int64(24*time.Hour/time.Second)),
			Type:    JobTypeBump,
			Timeout: bumpJobTimeout,
			Data:    string(data),
		}
		if err = s.sched.RunJob(ctx, def); err != nil {
			log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("failed to schedule shield bump job")
		}
	}
}

// Handler is the job handler of bump jobs.
type Handler struct{ s *Service }

// Handler returns the job handler.
func (s *Service) Handler() Handler { return Handler{s: s} }

// Handle implements job.Handler.
func (h Handler) Handle(ctx context.Context, input string, _ job.ProgressReporter) (string, error) {
	var in jobInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return "", fmt.Errorf("invalid shield bump job input: %w", err)
	}
	repo, err := h.s.repos.FindByID(ctx, in.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo %d: %w", in.RepoID, err)
	}
	f, err := h.s.findings.Find(ctx, in.FindingID)
	if err != nil {
		return "", fmt.Errorf("failed to find finding %d: %w", in.FindingID, err)
	}
	res, err := h.s.Bump(ctx, repo, f)
	if err != nil {
		// refusals that retrying cannot change are not job failures.
		if IsNeedsAgent(err) || errors.Is(err, ErrUnsupported) {
			return err.Error(), nil
		}
		return "", err
	}
	if res.Skipped != "" {
		return "skipped: " + res.Skipped, nil
	}
	return fmt.Sprintf("opened pull request #%d on %s", res.PRNum, res.Branch), nil
}
