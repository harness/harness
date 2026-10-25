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
	"strconv"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/require"
)

type fakeGuard struct {
	deny     bool
	checks   int
	attempts []agentguard.AttemptStatus
}

func (g *fakeGuard) Check(context.Context, agentguard.Request) agentguard.Decision {
	g.checks++
	if g.deny {
		return agentguard.Decision{Reason: agentguard.ReasonRepoKillSwitch, Detail: "off"}
	}
	return agentguard.Decision{Allowed: true, Reason: agentguard.ReasonAllowed}
}

func (g *fakeGuard) StartAttempt(context.Context, agentguard.Request) (agentguard.Attempt, error) {
	g.attempts = append(g.attempts, agentguard.AttemptRunning)
	return agentguard.Attempt{ID: int64(len(g.attempts))}, nil
}

func (g *fakeGuard) FinishAttempt(_ context.Context, id int64, s agentguard.AttemptStatus) error {
	g.attempts[id-1] = s
	return nil
}

type fakeSettings struct{ autonomy int }

func (f fakeSettings) AgentSettings(context.Context, int64) (*settings.AgentSettings, error) {
	return &settings.AgentSettings{
		KillSwitch: ptr.Bool(false), Autonomy: ptr.Int(f.autonomy), MaxActive: ptr.Int(3),
	}, nil
}

type fakeRepo struct {
	files    map[string]File
	branches map[string]bool
	commits  []commit
}

type commit struct {
	branch, message string
	files           []FileChange
}

func (r *fakeRepo) HeadSHA(context.Context, *types.RepositoryCore, string) (string, error) {
	return "head123", nil
}

func (r *fakeRepo) ReadFile(_ context.Context, _ *types.RepositoryCore, _, p string) (File, bool, error) {
	f, ok := r.files[p]
	return f, ok, nil
}

func (r *fakeRepo) BranchExists(_ context.Context, _ *types.RepositoryCore, b string) (bool, error) {
	return r.branches[b], nil
}

func (r *fakeRepo) CreateBranch(_ context.Context, _ *types.RepositoryCore, b, _ string) error {
	r.branches[b] = true
	return nil
}

func (r *fakeRepo) Commit(_ context.Context, _ *types.RepositoryCore, b, m string, fs []FileChange) error {
	r.commits = append(r.commits, commit{b, m, fs})
	return nil
}

type fakePRs struct {
	open    bool
	created []PullRequestInput
}

func (p *fakePRs) HasOpen(context.Context, *types.RepositoryCore, string, string) (bool, error) {
	return p.open, nil
}

func (p *fakePRs) Create(_ context.Context, _ *types.RepositoryCore, in PullRequestInput) (PullRequest, error) {
	p.created = append(p.created, in)
	return PullRequest{ID: 500, Number: 7}, nil
}

type fakeTasks struct {
	links    []string
	comments []string
}

func (t *fakeTasks) Find(context.Context, int64) (*types.Task, error) {
	return &types.Task{ID: 9, Number: 12}, nil
}

func (t *fakeTasks) Link(_ context.Context, _, _ int64, k enum.TaskLinkKind, ref string) error {
	t.links = append(t.links, string(k)+":"+ref)
	return nil
}

func (t *fakeTasks) AddComment(_ context.Context, _, _ int64, text string) error {
	t.comments = append(t.comments, text)
	return nil
}

type fakeSched struct{ defs []job.Definition }

func (s *fakeSched) RunJob(_ context.Context, d job.Definition) error {
	s.defs = append(s.defs, d)
	return nil
}

type env struct {
	svc   *Service
	guard *fakeGuard
	repo  *fakeRepo
	prs   *fakePRs
	tasks *fakeTasks
	sched *fakeSched
}

func newEnv(t *testing.T, autonomy int) *env {
	t.Helper()
	e := &env{
		guard: &fakeGuard{},
		repo:  &fakeRepo{files: map[string]File{}, branches: map[string]bool{}},
		prs:   &fakePRs{},
		tasks: &fakeTasks{},
		sched: &fakeSched{},
	}
	e.svc = NewService(Config{Enabled: true, PatchThreshold: 70}, e.guard, fakeSettings{autonomy},
		e.repo, e.prs, e.tasks, nil, nil, e.sched, func() *types.Principal { return &types.Principal{ID: 77} })
	return e
}

var testRepo = &types.RepositoryCore{ID: 1, Path: "space/repo", DefaultBranch: "main"}

func goFinding() *types.ShieldFinding {
	return &types.ShieldFinding{
		ID: 3, RepoID: 1, Ecosystem: "Go", Package: testMod, InstalledVersion: "v1.2.0",
		ManifestPath: "go.mod", VulnID: "GHSA-aaaa-bbbb", Aliases: []string{"CVE-2024-1"},
		Summary: "bad", RiskScore: 80, MinFixedVersion: "v1.2.5",
		FixStrategy: ptr.Of(enum.ShieldFixStrategyBump), State: enum.ShieldFindingStateOpen,
		TaskID: ptr.Int64(9),
	}
}

func (e *env) withGoFiles(t *testing.T) {
	proxy := fakeProxy(t, map[string]string{
		"v1.2.5": "module example.com/vuln\n\ngo 1.20\n\nrequire example.com/other v1.0.0\n",
	})
	e.svc.SetProxy(proxy)
	e.repo.files["go.mod"] = File{Content: []byte(projectMod), BlobSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	e.repo.files["go.sum"] = File{Content: []byte("example.com/other v1.0.0/go.mod h1:o=\n"), BlobSHA: "bbb"}
}

func TestBumpCreatesDraftPR(t *testing.T) {
	e := newEnv(t, settings.AutonomyDraftPR)
	e.withGoFiles(t)

	res, err := e.svc.Bump(context.Background(), testRepo, goFinding())
	require.NoError(t, err)
	require.Equal(t, "harness-agent/shield/GHSA-aaaa-bbbb-example.com-vuln", res.Branch)
	require.Equal(t, int64(7), res.PRNum)

	require.Len(t, e.repo.commits, 1)
	c := e.repo.commits[0]
	require.Equal(t, res.Branch, c.branch)
	require.Equal(t, "fix(deps): bump example.com/vuln to v1.2.5 (GHSA-aaaa-bbbb, CVE-2024-1)", c.message)
	require.Len(t, c.files, 2)
	require.Contains(t, string(c.files[0].Content), "example.com/vuln v1.2.5")

	require.Len(t, e.prs.created, 1)
	pr := e.prs.created[0]
	require.True(t, pr.IsDraft)
	require.Equal(t, "main", pr.TargetBranch)
	require.Contains(t, pr.Description, "Fixes TASK-12")
	require.Contains(t, pr.Description, "GHSA-aaaa-bbbb")

	require.Equal(t, []string{"pullreq:" + strconv.Itoa(500)}, e.tasks.links)
	require.Len(t, e.tasks.comments, 1)
	require.Equal(t, []agentguard.AttemptStatus{agentguard.AttemptSucceeded}, e.guard.attempts)
}

func TestBumpReadyPRAtAutoMerge(t *testing.T) {
	e := newEnv(t, settings.AutonomyAutoMerge)
	e.withGoFiles(t)
	_, err := e.svc.Bump(context.Background(), testRepo, goFinding())
	require.NoError(t, err)
	require.False(t, e.prs.created[0].IsDraft)
}

func TestBumpSkips(t *testing.T) {
	ctx := context.Background()

	t.Run("open pr", func(t *testing.T) {
		e := newEnv(t, 2)
		e.prs.open = true
		res, err := e.svc.Bump(ctx, testRepo, goFinding())
		require.NoError(t, err)
		require.NotEmpty(t, res.Skipped)
		require.Zero(t, e.guard.checks)
		require.Empty(t, e.repo.commits)
	})
	t.Run("guard denies", func(t *testing.T) {
		e := newEnv(t, 2)
		e.guard.deny = true
		res, err := e.svc.Bump(ctx, testRepo, goFinding())
		require.NoError(t, err)
		require.Contains(t, res.Skipped, string(agentguard.ReasonRepoKillSwitch))
		require.Empty(t, e.prs.created)
		require.Empty(t, e.guard.attempts)
	})
	t.Run("disabled", func(t *testing.T) {
		e := newEnv(t, 2)
		e.svc.config.Enabled = false
		res, err := e.svc.Bump(ctx, testRepo, goFinding())
		require.NoError(t, err)
		require.NotEmpty(t, res.Skipped)
	})
	t.Run("cross major is not eligible", func(t *testing.T) {
		e := newEnv(t, 2)
		f := goFinding()
		f.MinFixedVersion = "v2.0.0"
		res, err := e.svc.Bump(ctx, testRepo, f)
		require.NoError(t, err)
		require.NotEmpty(t, res.Skipped)
	})
}

func TestBumpNeedsAgent(t *testing.T) {
	e := newEnv(t, 2)
	e.withGoFiles(t)
	e.svc.SetProxy(fakeProxy(t, map[string]string{
		"v1.2.5": "module example.com/vuln\n\ngo 1.30\n",
	}))
	_, err := e.svc.Bump(context.Background(), testRepo, goFinding())
	require.True(t, IsNeedsAgent(err), err)
	require.Empty(t, e.repo.branches, "no branch must be created")
	require.Empty(t, e.prs.created)
	require.Len(t, e.tasks.comments, 1)
	require.Contains(t, e.tasks.comments[0], "fix agent is needed")
}

func TestBumpNpmUnsupported(t *testing.T) {
	e := newEnv(t, 2)
	e.repo.files["package-lock.json"] = File{Content: []byte("{}")}
	f := goFinding()
	f.Ecosystem, f.Package, f.ManifestPath, f.InstalledVersion, f.MinFixedVersion =
		"npm", "lodash", "package-lock.json", "4.17.0", "4.17.21"
	_, err := e.svc.Bump(context.Background(), testRepo, f)
	require.True(t, IsNeedsAgent(err), err)
}

func TestBumpRequirementsPR(t *testing.T) {
	e := newEnv(t, 2)
	e.repo.files["svc/requirements.txt"] = File{Content: []byte("django==3.2.1\n"), BlobSHA: "ccc"}
	f := goFinding()
	f.Ecosystem, f.Package, f.ManifestPath, f.InstalledVersion, f.MinFixedVersion =
		"PyPI", "django", "svc/requirements.txt", "3.2.1", "3.2.25"
	res, err := e.svc.Bump(context.Background(), testRepo, f)
	require.NoError(t, err)
	require.Equal(t, "harness-agent/shield/GHSA-aaaa-bbbb-django", res.Branch)
	require.Equal(t, "django==3.2.25\n", string(e.repo.commits[0].files[0].Content))
}

func TestBranchExistsWithoutPRFails(t *testing.T) {
	e := newEnv(t, 2)
	e.withGoFiles(t)
	e.repo.branches[BranchName(goFinding())] = true
	_, err := e.svc.Bump(context.Background(), testRepo, goFinding())
	require.Error(t, err)
	require.Equal(t, []agentguard.AttemptStatus{agentguard.AttemptFailed}, e.guard.attempts)
}

func TestOnFindingsEnqueuesEligible(t *testing.T) {
	e := newEnv(t, 2)
	low := goFinding()
	low.ID, low.RiskScore = 4, 10
	noTask := goFinding()
	noTask.ID, noTask.TaskID = 5, nil
	crossMajor := goFinding()
	crossMajor.ID, crossMajor.MinFixedVersion = 6, "v3.0.0"
	noFix := goFinding()
	noFix.ID, noFix.MinFixedVersion = 8, ""

	e.svc.OnFindings(context.Background(), testRepo, []*types.ShieldFinding{goFinding(), low, noTask, crossMajor, noFix})
	require.Len(t, e.sched.defs, 1)
	require.Equal(t, JobTypeBump, e.sched.defs[0].Type)
	require.True(t, strings.HasPrefix(e.sched.defs[0].UID, "shield_bump-1-3-"), e.sched.defs[0].UID)
	require.JSONEq(t, `{"repo_id":1,"finding_id":3}`, e.sched.defs[0].Data)
}

func TestBranchNameSanitized(t *testing.T) {
	f := goFinding()
	f.Package = "github.com/a/b..c.lock"
	f.VulnID = "CVE 2024/1"
	require.Equal(t, "harness-agent/shield/CVE-2024-1-github.com-a-b.c", BranchName(f))
}
