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

package agentguard

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// Reason explains the outcome of a guard decision.
type Reason string

const (
	ReasonAllowed            Reason = "allowed"
	ReasonGatewayDisabled    Reason = "ai_gateway_disabled"
	ReasonGlobalKillSwitch   Reason = "global_kill_switch"
	ReasonRepoKillSwitch     Reason = "repo_kill_switch"
	ReasonAutonomyTooLow     Reason = "autonomy_too_low"
	ReasonTriggeredByAgent   Reason = "triggered_by_agent"
	ReasonTooManyFailures    Reason = "too_many_failed_attempts"
	ReasonCooldown           Reason = "cooldown_active"
	ReasonMaxActiveAgents    Reason = "max_active_agents_reached"
	ReasonInternalError      Reason = "internal_error"
	ReasonAgentNotConfigured Reason = "agent_principal_not_configured"
)

// Settings provides the per repo agent settings.
type Settings interface {
	AgentSettings(ctx context.Context, repoID int64) (*settings.AgentSettings, error)
}

// ServiceSettings is a Settings backed by the settings service.
type ServiceSettings struct {
	Service *settings.Service
}

func (s ServiceSettings) AgentSettings(ctx context.Context, repoID int64) (*settings.AgentSettings, error) {
	return settings.RepoGetAgentSettings(ctx, s.Service, repoID)
}

// Config holds the global guard configuration.
type Config struct {
	// GatewayEnabled is true if the AI gateway is enabled. Agents are off otherwise.
	GatewayEnabled bool
	// KillSwitch disables all agents.
	KillSwitch bool
	// MaxFailedAttempts is the number of failed attempts after which a task is refused.
	MaxFailedAttempts int
	// Cooldown is the minimum time between attempts of the same task.
	Cooldown time.Duration
	// AttemptTimeout is the age after which a running attempt no longer counts as active. 0 disables it.
	AttemptTimeout time.Duration
}

const (
	defaultMaxFailedAttempts = 3
	defaultCooldown          = 30 * time.Minute
)

// NewConfig derives the guard configuration from the global config.
func NewConfig(global *types.Config) Config {
	return Config{
		GatewayEnabled:    global.AIGateway.Enabled,
		KillSwitch:        global.AIAgents.KillSwitch,
		MaxFailedAttempts: global.AIAgents.MaxFailedAttempts,
		Cooldown:          global.AIAgents.AttemptCooldown,
		AttemptTimeout:    global.AIAgents.AttemptTimeout,
	}
}

// Request asks whether an agent may act.
type Request struct {
	RepoID    int64
	SpacePath string // space path of the repo, used for the audit log
	RepoPath  string // full repo path, recorded in the audit log
	// TaskKey identifies the task the agent works on.
	TaskKey string
	// MinAutonomy is the autonomy level the action requires (e.g. settings.AutonomyDraftPR to open a PR).
	MinAutonomy int
	// TriggeredByPrincipalID is the principal that caused the triggering event (0 if unknown).
	TriggeredByPrincipalID int64
	// NoAI marks deterministic actions that use no AI (for example a dependency bump): they are not denied
	// because the AI gateway is disabled. Kill switches, autonomy and loop prevention still apply. The zero
	// value is an AI agent, so a caller that forgets the flag stays denied when the gateway is off.
	NoAI bool
	// RetryOtherModel marks the retry of a failed attempt with a different model: the cooldown does not
	// apply to it. The maximum number of failed attempts still does.
	RetryOtherModel bool
}

// Decision is the outcome of Check.
type Decision struct {
	Allowed bool
	Reason  Reason
	Detail  string
}

// Guard decides whether autonomous agents may act.
type Guard struct {
	config    Config
	settings  Settings
	attempts  AttemptStore
	audit     audit.Service
	principal func() *types.Principal
	now       func() time.Time
}

// New creates a Guard. principal returns the harness-agent service principal (nil if not bootstrapped).
func New(
	config Config,
	agentSettings Settings,
	attempts AttemptStore,
	auditService audit.Service,
	principal func() *types.Principal,
) *Guard {
	if config.MaxFailedAttempts <= 0 {
		config.MaxFailedAttempts = defaultMaxFailedAttempts
	}
	if config.Cooldown <= 0 {
		config.Cooldown = defaultCooldown
	}
	return &Guard{
		config:    config,
		settings:  agentSettings,
		attempts:  attempts,
		audit:     auditService,
		principal: principal,
		now:       time.Now,
	}
}

// Check answers whether an agent may act on the repo for the task now. Every decision is audited.
// Any internal failure results in a denial.
func (g *Guard) Check(ctx context.Context, req Request) Decision {
	d := g.decide(ctx, req)
	g.record(ctx, req, d)
	return d
}

// StartAttempt atomically reserves and records the start of an agent attempt for the task.
// See TryStartAttempt. A request marked RetryOtherModel skips the cooldown.
func (g *Guard) StartAttempt(ctx context.Context, req Request) (Attempt, error) {
	return g.TryStartAttempt(ctx, req, StartOptions{SkipCooldown: req.RetryOtherModel})
}

// FinishAttempt records the outcome of an attempt.
func (g *Guard) FinishAttempt(ctx context.Context, attemptID int64, status AttemptStatus) error {
	return g.attempts.Finish(ctx, attemptID, status)
}

func deny(reason Reason, format string, args ...any) Decision {
	return Decision{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

//nolint:gocognit // linear sequence of independent checks
func (g *Guard) decide(ctx context.Context, req Request) Decision {
	agent := g.principal()
	if agent == nil {
		return deny(ReasonAgentNotConfigured, "agent service principal is not set up")
	}
	if !req.NoAI && !g.config.GatewayEnabled {
		return deny(ReasonGatewayDisabled, "ai gateway is disabled")
	}
	if g.config.KillSwitch {
		return deny(ReasonGlobalKillSwitch, "agents are disabled globally")
	}

	s, err := g.settings.AgentSettings(ctx, req.RepoID)
	if err != nil {
		return deny(ReasonInternalError, "failed to load repo agent settings: %v", err)
	}
	if *s.KillSwitch {
		return deny(ReasonRepoKillSwitch, "agents are disabled for the repo")
	}
	if *s.Autonomy < req.MinAutonomy {
		return deny(ReasonAutonomyTooLow, "repo autonomy %d is below required %d", *s.Autonomy, req.MinAutonomy)
	}

	if req.TriggeredByPrincipalID != 0 && req.TriggeredByPrincipalID == agent.ID {
		return deny(ReasonTriggeredByAgent, "event was caused by the agent principal %q", agent.UID)
	}

	attempts, err := g.attempts.ListByTask(ctx, req.TaskKey)
	if err != nil {
		return deny(ReasonInternalError, "failed to load attempts: %v", err)
	}
	failed := 0
	var last time.Time
	for _, a := range attempts {
		if a.Status == AttemptFailed {
			failed++
		}
		if a.StartedAt.After(last) {
			last = a.StartedAt
		}
	}
	if failed >= g.config.MaxFailedAttempts {
		return deny(ReasonTooManyFailures, "task failed %d times (max %d)", failed, g.config.MaxFailedAttempts)
	}
	if !req.RetryOtherModel && !last.IsZero() && g.now().Sub(last) < g.config.Cooldown {
		return deny(ReasonCooldown, "last attempt started %s ago (cooldown %s)",
			g.now().Sub(last).Round(time.Second), g.config.Cooldown)
	}

	active, err := g.attempts.CountActive(ctx, req.RepoID)
	if err != nil {
		return deny(ReasonInternalError, "failed to count active agents: %v", err)
	}
	if active >= *s.MaxActive {
		return deny(ReasonMaxActiveAgents, "%d active agents (max %d)", active, *s.MaxActive)
	}

	return Decision{Allowed: true, Reason: ReasonAllowed}
}

func (g *Guard) record(ctx context.Context, req Request, d Decision) {
	agent := g.principal()
	if agent == nil {
		log.Ctx(ctx).Warn().Msgf("agent guard decision for task %q not audited: no agent principal", req.TaskKey)
		return
	}

	action := audit.ActionAllowed
	if !d.Allowed {
		action = audit.ActionDenied
	}

	err := g.audit.Log(ctx,
		*agent,
		audit.NewResource(audit.ResourceTypeAgentTask, req.TaskKey, audit.RepoPath, req.RepoPath),
		action,
		req.SpacePath,
		audit.WithData("reason", string(d.Reason), "detail", d.Detail),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to insert audit log for agent guard decision on task %q", req.TaskKey)
	}
}
