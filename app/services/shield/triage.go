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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// DefaultTriageThreshold is the risk score from which a finding is triaged by the AI.
	DefaultTriageThreshold = 40

	// minReachableConfidence is the confidence from which a reachable verdict raises the risk.
	minReachableConfidence = 0.5
	// highConfidence is the confidence from which not_reachable lowers the risk and a false
	// positive verdict is acted on.
	highConfidence = risk.DefaultHighConfidence
	// ruleAgentRisk is the risk from which the rules pick an agent fix when no same-line fix exists.
	ruleAgentRisk = 70
)

// Decider is the part of the decide service used by the triage.
type Decider interface {
	Decide(ctx context.Context, in aidecide.DecideInput) (*aidecide.Result, error)
}

// ModeResolver resolves the AI mode of a repository.
type ModeResolver interface {
	Resolve(ctx context.Context, repo *types.RepositoryCore) (enum.AIMode, error)
}

// AutonomyResolver returns the agent autonomy level (0-3) of a repository.
type AutonomyResolver func(ctx context.Context, repoID int64) (int, error)

// TriageStore is the finding storage used by the triage.
type TriageStore interface {
	List(ctx context.Context, repoID int64, filter *types.ShieldFindingFilter) ([]*types.ShieldFinding, error)
	UpdateState(ctx context.Context, id int64, state enum.ShieldFindingState, reason string, now int64) error
	UpdateTriage(ctx context.Context, id int64, in *types.ShieldFindingTriage, now int64) error
}

// TriageTasks is the subset of the task service used by the triage.
type TriageTasks interface {
	Update(ctx context.Context, principalID, taskID int64, in task.UpdateInput) (*types.Task, error)
	AddComment(ctx context.Context, principalID, taskID int64, text string) error
}

// AgentLauncher offers a finding to the fix agent. decisionID and confidence describe the decision that
// proposes the agent. It is implemented by agentlauncher.Launcher; the interface keeps the import graph acyclic.
type AgentLauncher interface {
	ConsiderFinding(ctx context.Context, repo *types.RepositoryCore, f *types.ShieldFinding, decisionID int64,
		confidence float64)
}

// TriageConfig configures the triage.
type TriageConfig struct {
	// Threshold is the risk score from which findings are triaged; zero means DefaultTriageThreshold.
	Threshold int
	// MaxFiles, MaxFileBytes and MaxTotalBytes bound the import search per ecosystem.
	MaxFiles      int
	MaxFileBytes  int64
	MaxTotalBytes int64
}

func (c TriageConfig) normalized() TriageConfig {
	if c.Threshold <= 0 {
		c.Threshold = DefaultTriageThreshold
	}
	if c.MaxFiles <= 0 {
		c.MaxFiles = 300
	}
	if c.MaxFileBytes <= 0 {
		c.MaxFileBytes = 64 << 10
	}
	if c.MaxTotalBytes <= 0 {
		c.MaxTotalBytes = 4 << 20
	}
	return c
}

// Triager asks the decision models about the Shield findings found by a scan: reachability, false
// positive and fix strategy. It refines the deterministic risk, never replaces it.
type Triager struct {
	config   TriageConfig
	svc      *Service
	findings TriageStore
	tasks    TriageTasks
	decider  Decider
	modes    ModeResolver
	autonomy AutonomyResolver
	agent    AgentLauncher
}

// SetAgentLauncher sets the launcher that is offered the findings whose fix strategy is the agent.
func (t *Triager) SetAgentLauncher(a AgentLauncher) { t.agent = a }

// NewTriager creates a triager over the scan service.
func NewTriager(
	config TriageConfig,
	svc *Service,
	findings TriageStore,
	tasks TriageTasks,
	decider Decider,
	modes ModeResolver,
	autonomy AutonomyResolver,
) *Triager {
	return &Triager{
		config: config.normalized(), svc: svc, findings: findings, tasks: tasks,
		decider: decider, modes: modes, autonomy: autonomy,
	}
}

// SettingsAutonomy returns an AutonomyResolver reading the repository agent settings.
func SettingsAutonomy(s *settings.Service) AutonomyResolver {
	return func(ctx context.Context, repoID int64) (int, error) {
		as, err := settings.RepoGetAgentSettings(ctx, s, repoID)
		if err != nil {
			return 0, err
		}
		return *as.Autonomy, nil
	}
}

// Triage triages the open findings seen by the scan. It stops without error when the AI quota is
// exhausted; the remaining findings stay untriaged and are picked up by the next scan.
func (t *Triager) Triage(ctx context.Context, repo *types.RepositoryCore, scan *types.ShieldScan) error {
	findings, err := t.collect(ctx, repo.ID, scan.ID)
	if err != nil || len(findings) == 0 {
		return err
	}
	exposure, err := t.svc.exposure(ctx, repo)
	if err != nil {
		return err
	}
	scorer := t.svc.data.Scorer()

	var todo []*types.ShieldFinding
	for _, f := range findings {
		if len(f.AIDecisionIDs) > 0 {
			t.reapply(ctx, repo, scorer, exposure, f)
			continue
		}
		todo = append(todo, f)
	}
	if len(todo) == 0 {
		return nil
	}

	mode, err := t.modes.Resolve(ctx, repo)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("repo_id", repo.ID).Msg("shield triage: failed to resolve ai mode")
		mode = enum.AIModeDecisionsOnly
	}
	autonomy := 0
	if t.autonomy != nil {
		if autonomy, err = t.autonomy(ctx, repo.ID); err != nil {
			log.Ctx(ctx).Warn().Err(err).Int64("repo_id", repo.ID).Msg("shield triage: failed to get autonomy")
			autonomy = 0
		}
	}

	run := &triageRun{
		t: t, repo: repo, scan: scan, mode: mode, autonomy: autonomy, scorer: scorer, exposure: exposure,
		direct: map[string]bool{}, parsed: map[string]bool{}, done: map[string]*triageOutcome{},
		imports: newImportIndex(t.svc.git, repo, scan.CommitSHA, t.config),
	}
	for _, f := range todo { // risk descending: the most important findings use the quota first
		if err = run.triage(ctx, f); err != nil {
			if isQuota(err) {
				log.Ctx(ctx).Info().Int64("repo_id", repo.ID).Int64("scan_id", scan.ID).
					Msg("shield triage: ai quota exhausted, leaving remaining findings for the next scan")
				return nil
			}
			log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("shield triage: finding failed")
		}
	}
	return nil
}

func isQuota(err error) bool {
	return errors.Is(err, aigateway.ErrQuotaExhausted) || aigateway.IsRateLimited(err)
}

// collect returns the open dependency findings of the scan at or above the threshold, risk descending.
func (t *Triager) collect(ctx context.Context, repoID, scanID int64) ([]*types.ShieldFinding, error) {
	var out []*types.ShieldFinding
	for page := 1; ; page++ {
		list, err := t.findings.List(ctx, repoID, &types.ShieldFindingFilter{
			Page: page, Size: pageSize, MinRiskScore: t.config.Threshold,
			States: []enum.ShieldFindingState{enum.ShieldFindingStateOpen},
			Kinds:  []enum.ShieldFindingKind{enum.ShieldFindingKindDependency},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list shield findings for triage: %w", err)
		}
		for _, f := range list {
			if f.LastSeenScanID == scanID {
				out = append(out, f)
			}
		}
		if len(list) < pageSize {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RiskScore > out[j].RiskScore })
	return out, nil
}

// reapply restores the risk of an already triaged finding: a rescan recomputes the deterministic
// risk with unknown reachability while the stored reachability is kept.
func (t *Triager) reapply(
	ctx context.Context,
	repo *types.RepositoryCore,
	scorer *risk.Scorer,
	exposure risk.Exposure,
	f *types.ShieldFinding,
) {
	if f.Reachable == enum.ShieldReachabilityUnknown {
		return
	}
	score := rescore(scorer, f, exposure, risk.Reachability(f.Reachable), 1)
	if score == f.RiskScore {
		return
	}
	old := f.RiskScore
	f.RiskScore = score
	if err := t.findings.UpdateTriage(ctx, f.ID, &types.ShieldFindingTriage{
		RiskScore: score, Reachable: f.Reachable, AIDecisionIDs: f.AIDecisionIDs,
	}, t.svc.now()); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("shield triage: failed to restore risk")
		return
	}
	t.syncTask(ctx, repo, f, old)
}

type triageOutcome struct {
	triage questions.ShieldTriage
	res    *aidecide.Result
}

type triageRun struct {
	t        *Triager
	repo     *types.RepositoryCore
	scan     *types.ShieldScan
	mode     enum.AIMode
	autonomy int
	scorer   *risk.Scorer
	exposure risk.Exposure
	imports  *importIndex

	direct map[string]bool // ecosystem|name|version|path -> direct
	parsed map[string]bool // manifest path -> parsed
	done   map[string]*triageOutcome
}

func triageKey(f *types.ShieldFinding) string {
	return f.VulnID + "|" + f.Ecosystem + "|" + f.Package + "|" + f.InstalledVersion
}

func (r *triageRun) triage(ctx context.Context, f *types.ShieldFinding) error {
	key := triageKey(f)
	out := r.done[key]
	if out == nil {
		res, err := r.t.decider.Decide(ctx, aidecide.DecideInput{
			TaskID:      f.TaskID,
			RepoID:      r.repo.ID,
			SessionID:   fmt.Sprintf("shield-%d-%d", r.repo.ID, r.scan.ID),
			State:       r.buildState(ctx, f),
			Questions:   questions.ShieldQuestions(),
			KeyQuestion: questions.QFixStrategy,
			Fallback:    rulesFallback(f),
		})
		if err != nil {
			return err
		}
		parsed, err := questions.ParseShield(res.Decisions)
		if err != nil {
			return fmt.Errorf("invalid shield triage decisions: %w", err)
		}
		out = &triageOutcome{triage: parsed, res: res}
		r.done[key] = out
	}
	return r.apply(ctx, f, out)
}

// apply stores the triage outcome on the finding and updates its task.
func (r *triageRun) apply(ctx context.Context, f *types.ShieldFinding, out *triageOutcome) error {
	t, res := out.triage, out.res
	reach, conf := enum.ShieldReachabilityUnknown, t.ReachableConfidence
	switch {
	case t.Reachable == questions.ReachabilityReachable && conf >= minReachableConfidence:
		reach = enum.ShieldReachabilityReachable
	case t.Reachable == questions.ReachabilityNotReachable && conf >= highConfidence:
		reach = enum.ShieldReachabilityNotReachable
	}

	oldRisk := f.RiskScore
	score := f.RiskScore
	if reach != enum.ShieldReachabilityUnknown {
		score = rescore(r.scorer, f, r.exposure, risk.Reachability(reach), conf)
	}

	var strategy *enum.ShieldFixStrategy
	if s, ok := mapStrategy(t.FixStrategy, f); ok {
		strategy = &s
	}

	qids := make([]string, 0, len(res.DecisionIDs))
	for q := range res.DecisionIDs {
		qids = append(qids, q)
	}
	sort.Strings(qids)
	ids := make([]int64, 0, len(qids))
	for _, q := range qids {
		ids = append(ids, res.DecisionIDs[q])
	}
	if len(ids) == 0 {
		// no audit rows (store unavailable): a marker is still needed to not triage again
		ids = []int64{0}
	}

	if err := r.t.findings.UpdateTriage(ctx, f.ID, &types.ShieldFindingTriage{
		RiskScore: score, Reachable: reach, FixStrategy: strategy, AIDecisionIDs: ids,
	}, r.t.svc.now()); err != nil {
		return fmt.Errorf("failed to store shield triage: %w", err)
	}
	f.RiskScore, f.Reachable, f.AIDecisionIDs = score, reach, ids
	if strategy != nil {
		f.FixStrategy = strategy
	}

	if t.FalsePositive && t.FalsePositiveConfidence >= highConfidence && !res.Rules &&
		r.autonomy >= settings.AutonomySuggest {
		reason := fmt.Sprintf("false positive (AI, confidence %.2f)", t.FalsePositiveConfidence)
		if err := r.t.findings.UpdateState(ctx, f.ID, enum.ShieldFindingStateIgnored, reason,
			r.t.svc.now()); err != nil {
			return fmt.Errorf("failed to ignore false positive: %w", err)
		}
		f.State, f.StateReason = enum.ShieldFindingStateIgnored, reason
		if f.TaskID != nil {
			if err := r.t.tasks.AddComment(ctx, r.t.svc.principalID(), *f.TaskID,
				"finding ignored: "+reason); err != nil {
				log.Ctx(ctx).Warn().Err(err).Int64("task_id", *f.TaskID).Msg("shield triage: failed to comment task")
			}
		}
		return nil
	}

	r.t.syncTask(ctx, r.repo, f, oldRisk)
	r.t.launchAgent(ctx, r.repo, f, strategy, res.DecisionIDs[questions.QFixStrategy], t.FixStrategyConfidence)
	return nil
}

// launchAgent offers an open finding with a task to the fix agent when the triage chose the agent strategy
// and its risk reaches the patch threshold.
func (t *Triager) launchAgent(
	ctx context.Context,
	repo *types.RepositoryCore,
	f *types.ShieldFinding,
	strategy *enum.ShieldFixStrategy,
	decisionID int64,
	confidence float64,
) {
	if t.agent == nil || strategy == nil || *strategy != enum.ShieldFixStrategyPatch ||
		f.State != enum.ShieldFindingStateOpen || f.TaskID == nil || f.RiskScore < t.svc.config.PatchThreshold {
		return
	}
	t.agent.ConsiderFinding(ctx, repo, f, decisionID, confidence)
}

// syncTask creates the task of a finding whose risk now reaches the patch threshold and moves the
// priority of an existing task when the risk crosses a priority threshold.
func (t *Triager) syncTask(ctx context.Context, repo *types.RepositoryCore, f *types.ShieldFinding, oldRisk int) {
	if f.State != enum.ShieldFindingStateOpen {
		return
	}
	if f.TaskID == nil {
		if err := t.svc.ensureTask(ctx, repo, f); err != nil {
			log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("shield triage: failed to create task")
		}
		return
	}
	if PriorityForRisk(f.RiskScore) == PriorityForRisk(oldRisk) {
		return
	}
	p := PriorityForRisk(f.RiskScore)
	if _, err := t.tasks.Update(ctx, t.svc.principalID(), *f.TaskID, task.UpdateInput{Priority: &p}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", *f.TaskID).Msg("shield triage: failed to update task priority")
	}
}

func rescore(
	scorer *risk.Scorer,
	f *types.ShieldFinding,
	exposure risk.Exposure,
	reach risk.Reachability,
	conf float64,
) int {
	score, _ := scorer.Score(risk.Input{
		CVSSScore:    f.CVSSScore,
		HasCVSS:      f.CVSSScore > 0,
		Severity:     severityOf(f.CVSSScore),
		CVEs:         cvesOf(f),
		Reachability: reach,
		Confidence:   conf,
		Exposure:     exposure,
	})
	return score
}

func severityOf(cvss float64) string {
	switch {
	case cvss >= 9:
		return "critical"
	case cvss >= 7:
		return "high"
	case cvss >= 4:
		return "medium"
	default:
		return "low"
	}
}

func cvesOf(f *types.ShieldFinding) []string {
	var out []string
	for _, id := range append([]string{f.VulnID}, f.Aliases...) {
		if strings.HasPrefix(strings.ToUpper(id), "CVE-") {
			out = append(out, id)
		}
	}
	return out
}

// mapStrategy maps the AI fix strategy to the finding enum. ok is false when the stored strategy is kept.
func mapStrategy(s questions.FixStrategy, f *types.ShieldFinding) (enum.ShieldFixStrategy, bool) {
	switch s {
	case questions.FixDeterministicBump:
		if f.MinFixedVersion == "" {
			return "", false
		}
		return enum.ShieldFixStrategyBump, true
	case questions.FixAgent:
		return enum.ShieldFixStrategyPatch, true
	case questions.FixManual:
		return enum.ShieldFixStrategyManual, true
	default: // accept_risk has no finding strategy; the decision stays in the audit trail
		return "", false
	}
}

// rulesFallback answers the Shield questions without AI: reachability unknown, never a false
// positive, a bump when a fix on the same major line exists, otherwise an agent for serious findings.
func rulesFallback(f *types.ShieldFinding) aidecide.FallbackFunc {
	return func(qs map[string]decisions.Question) map[string]decisions.Decision {
		strategy := questions.FixManual
		switch {
		case f.MinFixedVersion != "" && !isMajorBump(f.InstalledVersion, f.MinFixedVersion):
			strategy = questions.FixDeterministicBump
		case f.RiskScore >= ruleAgentRisk:
			strategy = questions.FixAgent
		}
		out := map[string]decisions.Decision{}
		for id, q := range qs {
			d := decisions.Decision{QuestionID: id, Type: q.Type, Confidence: 1}
			switch id {
			case questions.QReachable:
				d.Value = string(questions.ReachabilityUnknown)
			case questions.QFalsePositive:
				d.Value = "false"
			case questions.QFixStrategy:
				d.Value = string(strategy)
			}
			out[id] = d
		}
		return out
	}
}

// isMajorBump tells whether moving from one version to the other changes the first segment.
func isMajorBump(from, to string) bool {
	return majorOf(from) != majorOf(to)
}

func majorOf(v string) string {
	v = strings.TrimLeft(strings.TrimSpace(v), "vV")
	end := strings.IndexAny(v, ".-+")
	if end < 0 {
		end = len(v)
	}
	return v[:end]
}

// buildState builds the compact state of a finding. Decisions-only mode carries metadata and the
// import count only; full mode adds up to maxImportLines short import lines.
func (r *triageRun) buildState(ctx context.Context, f *types.ShieldFinding) *questions.ShieldState {
	st := &questions.ShieldState{
		FindingID:        strconv.FormatInt(f.ID, 10),
		Kind:             "vulnerability",
		Package:          f.Package,
		Ecosystem:        f.Ecosystem,
		Version:          f.InstalledVersion,
		FixedIn:          f.MinFixedVersion,
		CVE:              f.VulnID,
		CVSS:             f.CVSSScore,
		EPSS:             f.EPSS,
		KEV:              f.InKEV,
		Severity:         severityOf(f.CVSSScore),
		Summary:          f.Summary,
		Manifest:         f.ManifestPath,
		DirectDependency: r.isDirect(ctx, f),
		ImportCount:      -1,
	}
	if f.MinFixedVersion != "" {
		st.BumpIsMajor = isMajorBump(f.InstalledVersion, f.MinFixedVersion)
		st.DiffStats = "1 file, +1 -1"
	}
	if !aigateway.AllowsDecisions(r.mode) {
		return st // rules answer; nothing is sent
	}
	count, lines := r.imports.usages(ctx, f.Ecosystem, f.Package)
	st.ImportCount = count
	if r.mode == enum.AIModeFull {
		st.CallSites = lines
	}
	if err := st.Fit(questions.Limits{}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("finding_id", f.ID).Msg("shield triage: state does not fit")
	}
	return st
}

func (r *triageRun) isDirect(ctx context.Context, f *types.ShieldFinding) bool {
	if f.ManifestPath == "" {
		return false
	}
	if !r.parsed[f.ManifestPath] {
		r.parsed[f.ManifestPath] = true
		res, err := r.t.svc.parseManifest(ctx, git.ReadParams{RepoUID: r.repo.GitUID}, r.scan.CommitSHA,
			f.ManifestPath)
		if err != nil {
			log.Ctx(ctx).Debug().Err(err).Str("path", f.ManifestPath).Msg("shield triage: manifest not readable")
		} else {
			for _, p := range res.Packages {
				if p.Direct {
					r.direct[directKey(p)] = true
				}
			}
		}
	}
	return r.direct[directKey(osv.Package{
		Ecosystem: f.Ecosystem, Name: f.Package, Version: f.InstalledVersion, Path: f.ManifestPath,
	})]
}

func directKey(p osv.Package) string {
	return p.Ecosystem + "|" + p.Name + "|" + p.Version + "|" + p.Path
}

// ProvideTriager builds the triager and attaches it to the scan service. It is not part of WireSet
// directly: the server uses ProvideWiredTriager.
func ProvideTriager(
	config *types.Config,
	svc *Service,
	findings TriageStore,
	taskSvc TriageTasks,
	decide Decider,
	modes ModeResolver,
	settingsSvc *settings.Service,
) *Triager {
	t := NewTriager(TriageConfig{Threshold: config.Shield.TriageThreshold},
		svc, findings, taskSvc, decide, modes, SettingsAutonomy(settingsSvc))
	svc.SetTriager(t)
	return t
}
