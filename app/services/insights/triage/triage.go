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

// Package triage implements the insights.IssueObserver that turns new and regressed Insights
// issues into tasks: it gathers suspect commits and duplicate candidates, asks the decision
// model one batched question set, maps the answer to a task priority and records the decisions.
//
// Concurrency model: the observer callbacks only enqueue a job into a bounded channel drained by
// a small fixed pool of goroutines, so ingestion never waits for git or the model. When the queue
// is full the job is dropped with an error (logged by the insights service): backpressure instead
// of unbounded memory. The issue then has no task until it regresses or a later phase re-scans
// untriaged issues.
package triage

import (
	"context"
	"errors"
	"sync"

	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/insights/suspect"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	// ErrQuotaExhausted is returned by a Decider when the free quota is used up or rate limited.
	ErrQuotaExhausted = errors.New("ai quota exhausted")
	// ErrAIUnavailable is returned by a Decider when the AI is disabled or cannot answer.
	ErrAIUnavailable = errors.New("ai unavailable")
)

// DecideRequest is a batched set of questions about one state.
type DecideRequest struct {
	SessionID string
	State     any
	Questions map[string]decisions.Question
	// KeyQuestion drives confidence escalation.
	KeyQuestion string
}

// DecideResult is the answer of a Decider.
type DecideResult struct {
	Decisions map[string]decisions.Decision
	// Model is the exact model build that answered.
	Model    string
	Fallback bool
	// Human is true when the confidence of the key question was too low to act autonomously.
	Human bool
}

// Decider asks the decision model. Implementations map quota problems to ErrQuotaExhausted and a
// disabled or failing AI to ErrAIUnavailable (wrapped errors are fine).
type Decider interface {
	Decide(ctx context.Context, req DecideRequest) (*DecideResult, error)
}

// ModeResolver resolves the AI mode of a repository.
type ModeResolver interface {
	Resolve(ctx context.Context, repo *types.RepositoryCore) (enum.AIMode, error)
}

// RepoFinder loads repositories.
type RepoFinder interface {
	FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error)
}

// SuspectFinder computes ranked suspect commits for a repository.
type SuspectFinder interface {
	Find(ctx context.Context, repo *types.RepositoryCore, in suspect.Input) ([]suspect.Candidate, error)
}

// IssueStore is the part of the Insights issue store used here.
type IssueStore interface {
	Find(ctx context.Context, id int64) (*types.InsightsIssue, error)
	Update(ctx context.Context, issue *types.InsightsIssue) error
	List(ctx context.Context, repoID int64, filter *types.InsightsIssueFilter) ([]*types.InsightsIssue, error)
}

// EventStore is the part of the Insights event store used here.
type EventStore interface {
	ListByIssue(ctx context.Context, issueID int64, limit int) ([]*types.InsightsEvent, error)
}

// TaskService is the part of the task service used here.
type TaskService interface {
	Create(ctx context.Context, principalID int64, in task.CreateInput) (*types.Task, bool, error)
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
	Link(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error
	AddComment(ctx context.Context, principalID, taskID int64, text string) error
	RecordDecision(ctx context.Context, principalID, taskID int64, in task.DecisionInput) error
}

// AutofixObserver is told about tasks that the triage marked as autofix candidates (action=fix with enough
// confidence in a repository that allows agents). It decides by itself whether to launch an agent.
type AutofixObserver interface {
	AutofixCandidate(ctx context.Context, repo *types.RepositoryCore, t *types.Task, confidence float64)
}

// Config tunes the observer. Zero values pick defaults.
type Config struct {
	// Workers is the number of triage goroutines (default 2).
	Workers int
	// QueueSize bounds pending jobs (default 256).
	QueueSize int
	// DuplicateConfidence is the minimum probability and confidence to accept a duplicate (default 0.8).
	DuplicateConfidence float64
	// MaxScanIssues bounds the open issues scanned for duplicate candidates (default 50).
	MaxScanIssues int
}

func (c Config) withDefaults() Config {
	if c.Workers <= 0 {
		c.Workers = 2
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 256
	}
	if c.DuplicateConfidence <= 0 {
		c.DuplicateConfidence = 0.8
	}
	if c.MaxScanIssues <= 0 {
		c.MaxScanIssues = 50
	}
	return c
}

// Deps are the collaborators of the Observer.
type Deps struct {
	Issues   IssueStore
	Events   EventStore
	Repos    RepoFinder
	Modes    ModeResolver
	Suspects SuspectFinder
	Decider  Decider
	Tasks    TaskService
	// SystemPrincipal returns the id of the principal that owns automatic changes.
	SystemPrincipal func() int64
}

type job struct {
	project    *types.InsightsProject
	issue      *types.InsightsIssue
	event      *types.InsightsEvent
	regression bool
}

// Observer implements insights.IssueObserver.
type Observer struct {
	cfg     Config
	deps    Deps
	autofix AutofixObserver

	queue chan job
	wg    sync.WaitGroup
	once  sync.Once
	mu    sync.Mutex
	done  bool
}

var _ insights.IssueObserver = (*Observer)(nil)

// New creates an Observer. Call Start to launch the workers.
func New(cfg Config, deps Deps) *Observer {
	cfg = cfg.withDefaults()
	return &Observer{cfg: cfg, deps: deps, queue: make(chan job, cfg.QueueSize)}
}

// SetAutofixObserver registers the observer of autofix candidates (nil disables it). Safe to call while running.
func (o *Observer) SetAutofixObserver(a AutofixObserver) {
	o.mu.Lock()
	o.autofix = a
	o.mu.Unlock()
}

func (o *Observer) notifyAutofix(ctx context.Context, repo *types.RepositoryCore, t *types.Task, confidence float64) {
	o.mu.Lock()
	a := o.autofix
	o.mu.Unlock()
	if a != nil {
		a.AutofixCandidate(ctx, repo, t, confidence)
	}
}

// Start launches the workers; they stop when ctx is done or Stop is called.
func (o *Observer) Start(ctx context.Context) {
	o.once.Do(func() {
		for range o.cfg.Workers {
			o.wg.Add(1)
			go o.worker(ctx)
		}
	})
}

// Stop stops accepting jobs and waits for the workers to drain the queue.
func (o *Observer) Stop() {
	o.mu.Lock()
	if !o.done {
		o.done = true
		close(o.queue)
	}
	o.mu.Unlock()
	o.wg.Wait()
}

func (o *Observer) worker(ctx context.Context) {
	defer o.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j, ok := <-o.queue:
			if !ok {
				return
			}
			o.safeProcess(ctx, j)
		}
	}
}

func (o *Observer) safeProcess(ctx context.Context, j job) {
	defer func() {
		if r := recover(); r != nil {
			log.Ctx(ctx).Error().Interface("panic", r).Int64("issue_id", j.issue.ID).Msg("insights triage: panic")
		}
	}()
	if err := o.process(ctx, j); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("issue_id", j.issue.ID).Msg("insights triage: failed")
	}
}

// OnIssueCreated implements insights.IssueObserver.
func (o *Observer) OnIssueCreated(_ context.Context, p *types.InsightsProject, i *types.InsightsIssue,
	e *types.InsightsEvent) error {
	return o.enqueue(job{project: p, issue: i, event: e})
}

// OnIssueRegressed implements insights.IssueObserver.
func (o *Observer) OnIssueRegressed(_ context.Context, p *types.InsightsProject, i *types.InsightsIssue,
	e *types.InsightsEvent) error {
	return o.enqueue(job{project: p, issue: i, event: e, regression: true})
}

func (o *Observer) enqueue(j job) error {
	// snapshots: the caller may keep mutating its copies
	pc, ic, ec := *j.project, *j.issue, *j.event
	j.project, j.issue, j.event = &pc, &ic, &ec
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return errors.New("insights triage: observer stopped")
	}
	select {
	case o.queue <- j:
		return nil
	default:
		return errors.New("insights triage: queue full, issue not triaged")
	}
}
