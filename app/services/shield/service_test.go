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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	lockFixture  = "osv/testdata/lock/package-lock.json"
	cleanLock    = `{"lockfileVersion":3,"packages":{"":{"name":"x"}}}`
	recordsDir   = "osv/testdata/records"
	lodashFinder = "lodash"
)

// ---- fakes ----

type fakeScans struct {
	rows []*types.ShieldScan
}

func (f *fakeScans) Create(_ context.Context, s *types.ShieldScan) error {
	s.ID = int64(len(f.rows) + 1)
	c := *s
	f.rows = append(f.rows, &c)
	return nil
}

func (f *fakeScans) Update(_ context.Context, s *types.ShieldScan) error {
	c := *s
	f.rows[s.ID-1] = &c
	return nil
}

func (f *fakeScans) Find(_ context.Context, id int64) (*types.ShieldScan, error) {
	if id < 1 || int(id) > len(f.rows) {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *f.rows[id-1]
	return &c, nil
}

func (f *fakeScans) ListByRepo(_ context.Context, repoID int64, _, size int) ([]*types.ShieldScan, error) {
	var out []*types.ShieldScan
	for i := len(f.rows) - 1; i >= 0; i-- {
		if f.rows[i].RepoID == repoID && len(out) < size {
			c := *f.rows[i]
			out = append(out, &c)
		}
	}
	return out, nil
}

type fakeFindings struct {
	rows []*types.ShieldFinding
}

func (f *fakeFindings) Upsert(_ context.Context, in *types.ShieldFinding) error {
	for _, r := range f.rows {
		if r.RepoID == in.RepoID && r.Fingerprint == in.Fingerprint {
			state, reason, fixedAt := r.State, r.StateReason, r.FixedAt
			if r.State == enum.ShieldFindingStateFixed {
				state, reason, fixedAt = enum.ShieldFindingStateOpen, "reappeared", nil
			}
			first, firstScan, id, task, created := r.FirstSeen, r.FirstSeenScanID, r.ID, r.TaskID, r.Created
			*r = *in
			r.State, r.StateReason, r.FixedAt = state, reason, fixedAt
			r.FirstSeen, r.FirstSeenScanID, r.ID, r.TaskID, r.Created = first, firstScan, id, task, created
			*in = *r
			return nil
		}
	}
	c := *in
	c.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, &c)
	*in = c
	return nil
}

func (f *fakeFindings) Find(_ context.Context, id int64) (*types.ShieldFinding, error) {
	for _, r := range f.rows {
		if r.ID == id {
			c := *r
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *fakeFindings) match(r *types.ShieldFinding, repoID int64, filter *types.ShieldFindingFilter) bool {
	if r.RepoID != repoID {
		return false
	}
	if filter == nil {
		return true
	}
	if len(filter.States) > 0 {
		ok := false
		for _, s := range filter.States {
			ok = ok || s == r.State
		}
		if !ok {
			return false
		}
	}
	return r.RiskScore >= filter.MinRiskScore
}

func (f *fakeFindings) List(
	_ context.Context, repoID int64, filter *types.ShieldFindingFilter,
) ([]*types.ShieldFinding, error) {
	var all []*types.ShieldFinding
	for _, r := range f.rows {
		if f.match(r, repoID, filter) {
			c := *r
			all = append(all, &c)
		}
	}
	if filter == nil || filter.Size == 0 {
		return all, nil
	}
	start := (max(filter.Page, 1) - 1) * filter.Size
	if start >= len(all) {
		return nil, nil
	}
	return all[start:min(start+filter.Size, len(all))], nil
}

func (f *fakeFindings) Count(_ context.Context, repoID int64, filter *types.ShieldFindingFilter) (int64, error) {
	var n int64
	for _, r := range f.rows {
		if f.match(r, repoID, filter) {
			n++
		}
	}
	return n, nil
}

func (f *fakeFindings) MarkFixedNotSeenIn(_ context.Context, repoID, scanID, now int64) (int64, error) {
	var n int64
	for _, r := range f.rows {
		if r.RepoID == repoID && r.State == enum.ShieldFindingStateOpen && r.LastSeenScanID != scanID {
			r.State, r.FixedAt = enum.ShieldFindingStateFixed, &now
			n++
		}
	}
	return n, nil
}

func (f *fakeFindings) SetTask(_ context.Context, id int64, taskID *int64, _ int64) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.TaskID = taskID
		}
	}
	return nil
}

func (f *fakeFindings) SetFixStrategy(_ context.Context, id int64, s enum.ShieldFixStrategy, _ int64) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.FixStrategy = &s
		}
	}
	return nil
}

func (f *fakeFindings) UpdateState(
	_ context.Context, id int64, state enum.ShieldFindingState, reason string, _ int64,
) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.State, r.StateReason = state, reason
		}
	}
	return nil
}

type fakeTasks struct {
	tasks    []*types.Task
	links    map[int64][]string
	comments map[int64][]string
	creates  int
}

func newFakeTasks() *fakeTasks {
	return &fakeTasks{links: map[int64][]string{}, comments: map[int64][]string{}}
}

func (f *fakeTasks) Create(_ context.Context, _ int64, in task.CreateInput) (*types.Task, bool, error) {
	f.creates++
	for _, t := range f.tasks {
		if t.RepoID == in.RepoID && t.Origin == in.Origin && t.OriginRef == in.OriginRef {
			return t, false, nil
		}
	}
	t := &types.Task{
		ID: int64(len(f.tasks) + 1), RepoID: in.RepoID, Title: in.Title, Description: in.Description,
		Origin: in.Origin, OriginRef: in.OriginRef, Priority: in.Priority, State: enum.TaskStateOpen,
	}
	f.tasks = append(f.tasks, t)
	return t, true, nil
}

func (f *fakeTasks) Link(_ context.Context, _, taskID int64, kind enum.TaskLinkKind, refID string) error {
	key := string(kind) + ":" + refID
	for _, l := range f.links[taskID] {
		if l == key {
			return nil
		}
	}
	f.links[taskID] = append(f.links[taskID], key)
	return nil
}

func (f *fakeTasks) AddComment(_ context.Context, _, taskID int64, text string) error {
	f.comments[taskID] = append(f.comments[taskID], text)
	return nil
}

type fakeGit struct {
	files map[string]string // path -> content; nil value path missing
	fail  map[string]bool
}

func (g *fakeGit) ListPaths(context.Context, *git.ListPathsParams) (*git.ListPathsOutput, error) {
	out := &git.ListPathsOutput{Files: []string{"README.md", "src/main.go"}}
	for p := range g.files {
		out.Files = append(out.Files, p)
	}
	return out, nil
}

func (g *fakeGit) GetTreeNode(_ context.Context, p *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	return &git.GetTreeNodeOutput{Node: git.TreeNode{SHA: p.Path, Path: p.Path}}, nil
}

func (g *fakeGit) GetBlob(_ context.Context, p *git.GetBlobParams) (*git.GetBlobOutput, error) {
	if g.fail[p.SHA] {
		return nil, errors.New("boom")
	}
	c := g.files[p.SHA]
	return &git.GetBlobOutput{Size: int64(len(c)), ContentSize: int64(len(c)), Content: io.NopCloser(strings.NewReader(c))}, nil
}

type fakeData struct {
	db     *osv.DB
	scorer *risk.Scorer
}

func (d fakeData) LoadDB([]osv.Package) (*osv.DB, error) { return d.db, nil }
func (d fakeData) Scorer() *risk.Scorer                  { return d.scorer }

type fakeVisibility struct{ public bool }

func (v fakeVisibility) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return v.public, nil
}

type fakeRepos struct{ repo *types.RepositoryCore }

func (f fakeRepos) FindByID(context.Context, int64) (*types.RepositoryCore, error) {
	return f.repo, nil
}

type fakeScheduler struct{ jobs []job.Definition }

func (f *fakeScheduler) RunJob(_ context.Context, d job.Definition) error {
	f.jobs = append(f.jobs, d)
	return nil
}

func (f *fakeScheduler) AddRecurring(context.Context, string, string, string, time.Duration) error {
	return nil
}

// ---- harness ----

type env struct {
	svc      *Service
	scans    *fakeScans
	findings *fakeFindings
	tasks    *fakeTasks
	git      *fakeGit
	sched    *fakeScheduler
	repo     *types.RepositoryCore
}

func readFixture(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func newEnv(t *testing.T, threshold int, public bool) *env {
	t.Helper()
	db, err := osv.LoadDir(filepath.FromSlash(recordsDir))
	if err != nil {
		t.Fatal(err)
	}
	kev, err := risk.ParseKEV(strings.NewReader(
		`{"vulnerabilities":[{"cveID":"CVE-2020-8203"},{"cveID":"CVE-2021-23337"}]}`))
	if err != nil {
		t.Fatal(err)
	}

	e := &env{
		scans:    &fakeScans{},
		findings: &fakeFindings{},
		tasks:    newFakeTasks(),
		git:      &fakeGit{files: map[string]string{"package-lock.json": readFixture(t, lockFixture)}},
		sched:    &fakeScheduler{},
		repo:     &types.RepositoryCore{ID: 7, Path: "space/repo", GitUID: "uid", DefaultBranch: "main"},
	}
	e.svc = NewService(
		Config{Enabled: true, PatchThreshold: threshold},
		e.scans, e.findings, e.tasks, e.git,
		fakeData{db: db, scorer: &risk.Scorer{KEV: kev}},
		fakeVisibility{public: public}, fakeRepos{repo: e.repo}, e.sched,
	)
	e.svc.principalID = func() int64 { return 1 }
	var tick int64 = 1000
	e.svc.now = func() int64 { tick++; return tick }
	return e
}

func (e *env) scan(t *testing.T) *types.ShieldScan {
	t.Helper()
	s, err := e.svc.Scan(context.Background(), e.repo, "abc123", enum.ShieldScanTriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// ---- tests ----

func TestScan_FindingsAndTasks(t *testing.T) {
	e := newEnv(t, 1, true)
	s := e.scan(t)

	if s.Status != enum.ShieldScanStatusDone || s.Finished == nil {
		t.Fatalf("scan not done: %+v", s)
	}
	if s.FindingsCount == 0 || s.FindingsCount != len(e.findings.rows) || s.PackagesCount == 0 {
		t.Fatalf("bad counts: %+v findings=%d", s, len(e.findings.rows))
	}
	stored := e.scans.rows[s.ID-1]
	if stored.Status != enum.ShieldScanStatusDone {
		t.Fatalf("scan row not persisted: %+v", stored)
	}

	var lodash *types.ShieldFinding
	for _, f := range e.findings.rows {
		if f.RiskScore <= 0 {
			t.Errorf("finding %s has no risk", f.VulnID)
		}
		if f.TaskID == nil {
			t.Fatalf("finding %s has no task (threshold 1)", f.VulnID)
		}
		if f.Package == lodashFinder && f.VulnID == "GHSA-p6mc-m468-83gw" {
			lodash = f
		}
	}
	if lodash == nil {
		t.Fatal("lodash finding missing")
	}
	if !lodash.InKEV {
		t.Error("KEV flag not set from the risk data")
	}

	if len(e.tasks.tasks) != len(e.findings.rows) {
		t.Fatalf("tasks=%d findings=%d", len(e.tasks.tasks), len(e.findings.rows))
	}
	tk := e.tasks.tasks[*lodash.TaskID-1]
	if tk.Origin != enum.TaskOriginShield || tk.OriginRef != lodash.Fingerprint {
		t.Errorf("bad task origin: %+v", tk)
	}
	if want := "GHSA-p6mc-m468-83gw in lodash 4.17.15"; tk.Title != want {
		t.Errorf("title = %q, want %q", tk.Title, want)
	}
	if tk.Priority != PriorityForRisk(lodash.RiskScore) {
		t.Errorf("priority = %s", tk.Priority)
	}
	if got := e.tasks.links[tk.ID]; len(got) != 1 || got[0] != "shield_finding:"+strconv.FormatInt(lodash.ID, 10) {
		t.Errorf("links = %v", got)
	}
}

func TestScan_ThresholdNoTasks(t *testing.T) {
	e := newEnv(t, 101, true)
	e.scan(t)
	if len(e.findings.rows) == 0 {
		t.Fatal("expected findings")
	}
	if len(e.tasks.tasks) != 0 || e.tasks.creates != 0 {
		t.Fatalf("no task expected below the threshold, got %d", len(e.tasks.tasks))
	}
}

func TestScan_ThresholdSplitsFindings(t *testing.T) {
	e := newEnv(t, 1, true)
	e.scan(t)
	scores := map[string]int{}
	for _, f := range e.findings.rows {
		scores[f.Fingerprint] = f.RiskScore
	}
	// pick a threshold between the lowest and highest score.
	lo, hi := 101, -1
	for _, s := range scores {
		lo, hi = min(lo, s), max(hi, s)
	}
	if lo == hi {
		t.Skip("fixture scores are all equal")
	}

	e2 := newEnv(t, hi, true)
	e2.scan(t)
	for _, f := range e2.findings.rows {
		if (f.RiskScore >= hi) != (f.TaskID != nil) {
			t.Errorf("finding %s score %d task=%v with threshold %d", f.VulnID, f.RiskScore, f.TaskID, hi)
		}
	}
}

func TestScan_PrivateRepoLowersRisk(t *testing.T) {
	pub := newEnv(t, 101, true)
	pub.scan(t)
	priv := newEnv(t, 101, false)
	priv.scan(t)

	byFP := map[string]int{}
	for _, f := range pub.findings.rows {
		byFP[f.Fingerprint] = f.RiskScore
	}
	lower := false
	for _, f := range priv.findings.rows {
		if f.RiskScore > byFP[f.Fingerprint] {
			t.Errorf("private score %d above public %d", f.RiskScore, byFP[f.Fingerprint])
		}
		lower = lower || f.RiskScore < byFP[f.Fingerprint]
	}
	if !lower {
		t.Error("exposure factor had no effect")
	}
}

func TestScan_RescanDedup(t *testing.T) {
	e := newEnv(t, 1, true)
	first := e.scan(t)
	nFindings, nTasks := len(e.findings.rows), len(e.tasks.tasks)

	second := e.scan(t)
	if len(e.findings.rows) != nFindings || len(e.tasks.tasks) != nTasks {
		t.Fatalf("rescan duplicated: findings %d->%d tasks %d->%d",
			nFindings, len(e.findings.rows), nTasks, len(e.tasks.tasks))
	}
	for _, f := range e.findings.rows {
		if f.FirstSeenScanID != first.ID || f.LastSeenScanID != second.ID || f.State != enum.ShieldFindingStateOpen {
			t.Errorf("finding %d: first=%d last=%d state=%s", f.ID, f.FirstSeenScanID, f.LastSeenScanID, f.State)
		}
	}
	for id, links := range e.tasks.links {
		if len(links) != 1 {
			t.Errorf("task %d has %d links", id, len(links))
		}
	}
	for id, c := range e.tasks.comments {
		t.Errorf("unexpected comment on task %d: %v", id, c)
	}
}

func TestScan_FixedDetection(t *testing.T) {
	e := newEnv(t, 1, true)
	e.scan(t)
	n := len(e.findings.rows)

	e.git.files["package-lock.json"] = cleanLock
	second := e.scan(t)
	if second.Status != enum.ShieldScanStatusDone || second.FindingsCount != 0 {
		t.Fatalf("second scan: %+v", second)
	}
	want := fmt.Sprintf("finding no longer detected in scan %d", second.ID)
	for _, f := range e.findings.rows {
		if f.State != enum.ShieldFindingStateFixed || f.FixedAt == nil {
			t.Errorf("finding %d not fixed: %s", f.ID, f.State)
		}
		c := e.tasks.comments[*f.TaskID]
		if len(c) != 1 || c[0] != want {
			t.Errorf("comments of task %d = %v", *f.TaskID, c)
		}
	}
	if len(e.tasks.comments) != n {
		t.Errorf("comments on %d tasks, want %d", len(e.tasks.comments), n)
	}

	// the vulnerability reappears: findings reopen, no new task is created for the same fingerprint.
	e.git.files["package-lock.json"] = readFixture(t, lockFixture)
	e.scan(t)
	for _, f := range e.findings.rows {
		if f.State != enum.ShieldFindingStateOpen {
			t.Errorf("finding %d not reopened", f.ID)
		}
	}
	if len(e.tasks.tasks) != n {
		t.Errorf("tasks = %d, want %d", len(e.tasks.tasks), n)
	}
}

func TestScan_IgnoredFindingNoTask(t *testing.T) {
	e := newEnv(t, 1, true)
	e.scan(t)
	for _, f := range e.findings.rows {
		f.State, f.TaskID = enum.ShieldFindingStateIgnored, nil
	}
	e.tasks.tasks, e.tasks.creates = nil, 0

	e.scan(t)
	if e.tasks.creates != 0 {
		t.Errorf("ignored findings must not create tasks, got %d", e.tasks.creates)
	}
}

func TestScan_UnreadableManifestKeepsFindings(t *testing.T) {
	e := newEnv(t, 1, true)
	e.scan(t)

	e.git.fail = map[string]bool{"package-lock.json": true}
	s := e.scan(t)
	if s.Status != enum.ShieldScanStatusDone {
		t.Fatalf("status = %s", s.Status)
	}
	for _, f := range e.findings.rows {
		if f.State != enum.ShieldFindingStateOpen {
			t.Errorf("finding %d marked %s although its manifest could not be read", f.ID, f.State)
		}
	}
}

func TestScan_Disabled(t *testing.T) {
	e := newEnv(t, 1, true)
	e.svc.config.Enabled = false
	if _, err := e.svc.Scan(context.Background(), e.repo, "abc", enum.ShieldScanTriggerManual); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("err = %v", err)
	}
	if len(e.scans.rows) != 0 {
		t.Fatal("no scan expected")
	}
}

func TestPriorityForRisk(t *testing.T) {
	for score, want := range map[int]enum.TaskPriority{
		100: enum.TaskPriorityP0, 90: enum.TaskPriorityP0, 89: enum.TaskPriorityP1, 70: enum.TaskPriorityP1,
		69: enum.TaskPriorityP2, 40: enum.TaskPriorityP2, 39: enum.TaskPriorityP3, 0: enum.TaskPriorityP3,
	} {
		if got := PriorityForRisk(score); got != want {
			t.Errorf("PriorityForRisk(%d) = %s, want %s", score, got, want)
		}
	}
}

func TestSyncCron(t *testing.T) {
	for in, want := range map[time.Duration]string{
		time.Minute: "0 * * * *", time.Hour: "0 * * * *", 6 * time.Hour: "0 */6 * * *",
		90 * time.Minute: "0 */2 * * *", 48 * time.Hour: "0 0 * * *",
	} {
		if got := syncCron(in); got != want {
			t.Errorf("syncCron(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestEnqueueAndJobHandler(t *testing.T) {
	e := newEnv(t, 1, true)
	scan, err := e.svc.Enqueue(context.Background(), e.repo, "abc123", enum.ShieldScanTriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Status != enum.ShieldScanStatusPending || len(e.sched.jobs) != 1 {
		t.Fatalf("scan=%+v jobs=%d", scan, len(e.sched.jobs))
	}
	def := e.sched.jobs[0]
	if def.Type != JobTypeScan || !strings.Contains(def.UID, "abc123") || def.Validate() != nil {
		t.Fatalf("bad job: %+v (%v)", def, def.Validate())
	}

	res, err := scanHandler{s: e.svc}.Handle(context.Background(), def.Data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, "done") {
		t.Errorf("result = %q", res)
	}
	if e.scans.rows[scan.ID-1].Status != enum.ShieldScanStatusDone {
		t.Fatalf("scan not run: %+v", e.scans.rows[scan.ID-1])
	}
}

func TestOnPush(t *testing.T) {
	e := newEnv(t, 1, true)
	ctx := context.Background()

	if err := e.svc.onPush(ctx, 7, "refs/heads/feature", "sha1"); err != nil || len(e.sched.jobs) != 0 {
		t.Fatalf("non default branch scanned: err=%v jobs=%d", err, len(e.sched.jobs))
	}
	if err := e.svc.onPush(ctx, 7, "refs/heads/main", "sha1"); err != nil || len(e.sched.jobs) != 1 {
		t.Fatalf("default branch not scheduled: err=%v jobs=%d", err, len(e.sched.jobs))
	}
	if e.scans.rows[0].Trigger != enum.ShieldScanTriggerPush || e.scans.rows[0].CommitSHA != "sha1" {
		t.Errorf("scan = %+v", e.scans.rows[0])
	}
	// replayed event
	if err := e.svc.onPush(ctx, 7, "refs/heads/main", "sha1"); err != nil || len(e.sched.jobs) != 1 {
		t.Fatalf("replay scheduled twice: err=%v jobs=%d", err, len(e.sched.jobs))
	}
	if err := e.svc.onPush(ctx, 7, "refs/heads/main", "sha2"); err != nil || len(e.sched.jobs) != 2 {
		t.Fatalf("new commit not scheduled: err=%v jobs=%d", err, len(e.sched.jobs))
	}

	e.svc.config.Enabled = false
	if err := e.svc.onPush(ctx, 7, "refs/heads/main", "sha3"); err != nil || len(e.sched.jobs) != 2 {
		t.Fatalf("disabled service scheduled: err=%v jobs=%d", err, len(e.sched.jobs))
	}
}
