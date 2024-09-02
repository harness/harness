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

package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	events "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/livelog"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// pipelineJWTLifetime specifies the max lifetime of an ephemeral pipeline jwt token.
	pipelineJWTLifetime = 72 * time.Hour
	// pipelineJWTRole specifies the role of an ephemeral pipeline jwt token.
	pipelineJWTRole = enum.MembershipRoleContributor
)

var noContext = context.Background()

var _ ExecutionManager = (*Manager)(nil)

type (
	// Request provides filters when requesting a pending
	// build from the queue. This allows an agent, for example,
	// to request a build that matches its architecture and kernel.
	Request struct {
		Kind    string            `json:"kind"`
		Type    string            `json:"type"`
		OS      string            `json:"os"`
		Arch    string            `json:"arch"`
		Variant string            `json:"variant"`
		Kernel  string            `json:"kernel"`
		Labels  map[string]string `json:"labels,omitempty"`
	}

	// Config represents a pipeline config file.
	Config struct {
		Data string `json:"data"`
		Kind string `json:"kind"`
	}

	// Netrc contains login and initialization information used
	// by an automated login process.
	Netrc struct {
		Machine  string `json:"machine"`
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	// ExecutionContext represents the minimum amount of information
	// required by the runner to execute a build.
	ExecutionContext struct {
		Repo         *types.Repository `json:"repository"`
		RepoIsPublic bool              `json:"repository_is_public,omitempty"`
		Execution    *types.Execution  `json:"build"`
		Stage        *types.Stage      `json:"stage"`
		Secrets      []*types.Secret   `json:"secrets"`
		Config       *file.File        `json:"config"`
		Netrc        *Netrc            `json:"netrc"`
	}

	// ExecutionManager encapsulates complex build operations and provides
	// a simplified interface for build runners.
	ExecutionManager interface {
		// Request requests the next available build stage for execution.
		Request(ctx context.Context, args *Request) (*types.Stage, error)

		// Watch watches for build cancellation requests.
		Watch(ctx context.Context, executionID int64) (bool, error)

		// Accept accepts the build stage for execution.
		Accept(ctx context.Context, stage int64, machine string) (*types.Stage, error)

		// Write writes a line to the build logs.
		Write(ctx context.Context, step int64, line *livelog.Line) error

		// Details returns details about stage.
		Details(ctx context.Context, stageID int64) (*ExecutionContext, error)

		// UploadLogs uploads the full logs.
		UploadLogs(ctx context.Context, step int64, r io.Reader) error

		// BeforeStep signals the build step is about to start.
		BeforeStep(ctx context.Context, step *types.Step) error

		// AfterStep signals the build step is complete.
		AfterStep(ctx context.Context, step *types.Step) error

		// BeforeStage signals the build stage is about to start.
		BeforeStage(ctx context.Context, stage *types.Stage) error

		// AfterStage signals the build stage is complete.
		AfterStage(ctx context.Context, stage *types.Stage) error
	}
)

// Manager provides a simplified interface to the build runner so that it
// can more easily interact with the server.
type Manager struct {
	Executions       store.ExecutionStore
	Config           *types.Config
	FileService      file.Service
	ConverterService converter.Service
	Pipelines        store.PipelineStore
	urlProvider      urlprovider.Provider
	Checks           store.CheckStore
	// Converter  store.ConvertService
	SSEStreamer sse.Streamer
	// Globals    store.GlobalSecretStore
	Logs store.LogStore
	Logz livelog.LogStream
	// Netrcs     store.NetrcService
	Repos     store.RepoStore
	Scheduler scheduler.Scheduler
	Secrets   store.SecretStore
	// Status  store.StatusService
	Stages store.StageStore
	Steps  store.StepStore
	// System  *store.System
	Users store.PrincipalStore
	// Webhook store.WebhookSender

	publicAccess publicaccess.Service
	// events reporter
	reporter events.Reporter
}

func New(
	config *types.Config,
	executionStore store.ExecutionStore,
	pipelineStore store.PipelineStore,
	urlProvider urlprovider.Provider,
	sseStreamer sse.Streamer,
	fileService file.Service,
	converterService converter.Service,
	logStore store.LogStore,
	logStream livelog.LogStream,
	checkStore store.CheckStore,
	repoStore store.RepoStore,
	scheduler scheduler.Scheduler,
	secretStore store.SecretStore,
	stageStore store.StageStore,
	stepStore store.StepStore,
	userStore store.PrincipalStore,
	publicAccess publicaccess.Service,
	reporter events.Reporter,
) *Manager {
	return &Manager{
		Config:           config,
		Executions:       executionStore,
		Pipelines:        pipelineStore,
		urlProvider:      urlProvider,
		SSEStreamer:      sseStreamer,
		FileService:      fileService,
		ConverterService: converterService,
		Logs:             logStore,
		Logz:             logStream,
		Checks:           checkStore,
		Repos:            repoStore,
		Scheduler:        scheduler,
		Secrets:          secretStore,
		Stages:           stageStore,
		Steps:            stepStore,
		Users:            userStore,
		publicAccess:     publicAccess,
		reporter:         reporter,
	}
}

// Request requests the next available build stage for execution.
func (m *Manager) Request(ctx context.Context, args *Request) (*types.Stage, error) {
	log := log.With().
		Str("kind", args.Kind).
		Str("type", args.Type).
		Str("os", args.OS).
		Str("arch", args.Arch).
		Str("kernel", args.Kernel).
		Str("variant", args.Variant).
		Logger()
	log.Debug().Msg("manager: request queue item")

	stage, err := m.Scheduler.Request(ctx, scheduler.Filter{
		Kind:    args.Kind,
		Type:    args.Type,
		OS:      args.OS,
		Arch:    args.Arch,
		Kernel:  args.Kernel,
		Variant: args.Variant,
		Labels:  args.Labels,
	})
	if err != nil && ctx.Err() != nil {
		log.Debug().Err(err).Msg("manager: context canceled")
		return nil, err
	}
	if err != nil {
		log.Warn().Err(err).Msg("manager: request queue item error")
		return nil, err
	}
	return stage, nil
}

// Accept accepts the build stage for execution. It is possible for multiple
// agents to pull the same stage from the queue.
func (m *Manager) Accept(_ context.Context, id int64, machine string) (*types.Stage, error) {
	log := log.With().
		Int64("stage-id", id).
		Str("machine", machine).
		Logger()
	log.Debug().Msg("manager: accept stage")

	stage, err := m.Stages.Find(noContext, id)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot find stage")
		return nil, err
	}
	if stage.Machine != "" {
		log.Debug().Msg("manager: stage already assigned. abort.")
		return nil, fmt.Errorf("stage already assigned, abort")
	}

	stage.Machine = machine
	stage.Status = enum.CIStatusPending
	err = m.Stages.Update(noContext, stage)
	switch {
	case errors.Is(err, gitness_store.ErrVersionConflict):
		log.Debug().Err(err).Msg("manager: stage processed by another agent")
	case err != nil:
		log.Debug().Err(err).Msg("manager: cannot update stage")
	default:
		log.Info().Msg("manager: stage accepted")
	}
	return stage, err
}

// Write writes a line to the build logs.
func (m *Manager) Write(ctx context.Context, step int64, line *livelog.Line) error {
	err := m.Logz.Write(ctx, step, line)
	if err != nil {
		log.Warn().Int64("step-id", step).Err(err).Msg("manager: cannot write to log stream")
		return err
	}
	return nil
}

// UploadLogs uploads the full logs.
func (m *Manager) UploadLogs(ctx context.Context, step int64, r io.Reader) error {
	err := m.Logs.Create(ctx, step, r)
	if err != nil {
		log.Error().Err(err).Int64("step-id", step).Msg("manager: cannot upload complete logs")
		return err
	}
	return nil
}

// Details provides details about the stage.
func (m *Manager) Details(ctx context.Context, stageID int64) (*ExecutionContext, error) {
	log := log.With().
		Int64("stage-id", stageID).
		Logger()
	log.Debug().Msg("manager: fetching stage details")

	stage, err := m.Stages.Find(noContext, stageID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot find stage")
		return nil, err
	}
	execution, err := m.Executions.Find(noContext, stage.ExecutionID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot find build")
		return nil, err
	}
	pipeline, err := m.Pipelines.Find(noContext, execution.PipelineID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot find pipeline")
		return nil, err
	}
	repo, err := m.Repos.Find(noContext, execution.RepoID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot find repo")
		return nil, err
	}

	// Backfill clone URL
	repo.GitURL = m.urlProvider.GenerateContainerGITCloneURL(ctx, repo.Path)

	stages, err := m.Stages.List(noContext, stage.ExecutionID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot list stages")
		return nil, err
	}
	execution.Stages = stages
	log = log.With().
		Int64("build", execution.Number).
		Str("repo", repo.GetGitUID()).
		Logger()

	// TODO: Currently we fetch all the secrets from the same space.
	// This logic can be updated when needed.
	secrets, err := m.Secrets.ListAll(noContext, repo.ParentID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot list secrets")
		return nil, err
	}

	// Fetch contents of YAML from the execution ref at the pipeline config path.
	file, err := m.FileService.Get(noContext, repo, pipeline.ConfigPath, execution.After)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot fetch file")
		return nil, err
	}

	// Get public access settings of the repo
	repoIsPublic, err := m.publicAccess.Get(noContext, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot check if repo is public")
		return nil, err
	}

	// Convert file contents in case templates are being used.
	args := &converter.ConvertArgs{
		Repo:         repo,
		Pipeline:     pipeline,
		Execution:    execution,
		File:         file,
		RepoIsPublic: repoIsPublic,
	}
	file, err = m.ConverterService.Convert(noContext, args)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot convert template contents")
		return nil, err
	}

	netrc, err := m.createNetrc(repo)
	if err != nil {
		log.Warn().Err(err).Msg("manager: failed to create netrc")
		return nil, err
	}

	return &ExecutionContext{
		Repo:         repo,
		RepoIsPublic: repoIsPublic,
		Execution:    execution,
		Stage:        stage,
		Secrets:      secrets,
		Config:       file,
		Netrc:        netrc,
	}, nil
}

func (m *Manager) createNetrc(repo *types.Repository) (*Netrc, error) {
	pipelinePrincipal := bootstrap.NewPipelineServiceSession().Principal
	jwt, err := jwt.GenerateWithMembership(
		pipelinePrincipal.ID,
		repo.ParentID,
		pipelineJWTRole,
		pipelineJWTLifetime,
		pipelinePrincipal.Salt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt: %w", err)
	}

	cloneURL, err := url.Parse(repo.GitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clone url '%s': %w", cloneURL, err)
	}

	return &Netrc{
		Machine:  cloneURL.Hostname(),
		Login:    pipelinePrincipal.UID,
		Password: jwt,
	}, nil
}

// Before signals the build step is about to start.
func (m *Manager) BeforeStep(_ context.Context, step *types.Step) error {
	log := log.With().
		Str("step.status", string(step.Status)).
		Str("step.name", step.Name).
		Int64("step.id", step.ID).
		Logger()

	log.Debug().Msg("manager: updating step status")

	err := m.Logz.Create(noContext, step.ID)
	if err != nil {
		log.Warn().Err(err).Msg("manager: cannot create log stream")
		return err
	}
	updater := &updater{
		Executions:  m.Executions,
		SSEStreamer: m.SSEStreamer,
		Repos:       m.Repos,
		Steps:       m.Steps,
		Stages:      m.Stages,
	}
	return updater.do(noContext, step)
}

// After signals the build step is complete.
func (m *Manager) AfterStep(_ context.Context, step *types.Step) error {
	log := log.With().
		Str("step.status", string(step.Status)).
		Str("step.name", step.Name).
		Int64("step.id", step.ID).
		Logger()
	log.Debug().Msg("manager: updating step status")

	var retErr error
	updater := &updater{
		Executions:  m.Executions,
		SSEStreamer: m.SSEStreamer,
		Repos:       m.Repos,
		Steps:       m.Steps,
		Stages:      m.Stages,
	}

	if err := updater.do(noContext, step); err != nil {
		retErr = err
		log.Warn().Err(err).Msg("manager: cannot update step")
	}

	if err := m.Logz.Delete(noContext, step.ID); err != nil && !errors.Is(err, livelog.ErrStreamNotFound) {
		log.Warn().Err(err).Msg("manager: cannot teardown log stream")
	}
	return retErr
}

// BeforeStage signals the build stage is about to start.
func (m *Manager) BeforeStage(_ context.Context, stage *types.Stage) error {
	s := &setup{
		Executions:  m.Executions,
		Checks:      m.Checks,
		Pipelines:   m.Pipelines,
		SSEStreamer: m.SSEStreamer,
		Repos:       m.Repos,
		Steps:       m.Steps,
		Stages:      m.Stages,
		Users:       m.Users,
	}

	return s.do(noContext, stage)
}

// AfterStage signals the build stage is complete.
func (m *Manager) AfterStage(_ context.Context, stage *types.Stage) error {
	t := &teardown{
		Executions:  m.Executions,
		Pipelines:   m.Pipelines,
		Checks:      m.Checks,
		SSEStreamer: m.SSEStreamer,
		Logs:        m.Logz,
		Repos:       m.Repos,
		Scheduler:   m.Scheduler,
		Steps:       m.Steps,
		Stages:      m.Stages,
		Reporter:    m.reporter,
	}
	return t.do(noContext, stage)
}

// Watch watches for build cancellation requests.
func (m *Manager) Watch(ctx context.Context, executionID int64) (bool, error) {
	ok, err := m.Scheduler.Cancelled(ctx, executionID)
	// we expect a context cancel error here which
	// indicates a polling timeout. The subscribing
	// client should look for the context cancel error
	// and resume polling.
	if err != nil {
		return ok, err
	}

	// // TODO: we should be able to return
	// // immediately if Cancelled returns true. This requires
	// // some more testing but would avoid the extra database
	// // call.
	// if ok {
	// 	return ok, err
	// }

	// if no error is returned we should check
	// the database to see if the build is complete. If
	// complete, return true.
	execution, err := m.Executions.Find(ctx, executionID)
	if err != nil {
		log := log.With().
			Int64("execution.id", executionID).
			Logger()
		log.Warn().Msg("manager: cannot find build")
		return ok, fmt.Errorf("could not find build for cancellation: %w", err)
	}
	return execution.Status.IsDone(), nil
}
