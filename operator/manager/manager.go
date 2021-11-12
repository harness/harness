// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/drone/drone-yaml/yaml/converter"
	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

var noContext = context.Background()

var _ BuildManager = (*Manager)(nil)

type (
	// Context represents the minimum amount of information
	// required by the runner to execute a build.
	Context struct {
		Repo    *core.Repository `json:"repository"`
		Build   *core.Build      `json:"build"`
		Stage   *core.Stage      `json:"stage"`
		Config  *core.File       `json:"config"`
		Secrets []*core.Secret   `json:"secrets"`
		System  *core.System     `json:"system"`
	}

	// BuildManager encapsulates complex build operations and provides
	// a simplified interface for build runners.
	BuildManager interface {
		// Request requests the next available build stage for execution.
		Request(ctx context.Context, args *Request) (*core.Stage, error)

		// Accept accepts the build stage for execution.
		Accept(ctx context.Context, stage int64, machine string) (*core.Stage, error)

		// Netrc returns a valid netrc for execution.
		Netrc(ctx context.Context, repo int64) (*core.Netrc, error)

		// Details fetches build details
		Details(ctx context.Context, stage int64) (*Context, error)

		// Before signals the build step is about to start.
		Before(ctx context.Context, step *core.Step) error

		// After signals the build step is complete.
		After(ctx context.Context, step *core.Step) error

		// BeforeAll signals the build stage is about to start.
		BeforeAll(ctx context.Context, stage *core.Stage) error

		// AfterAll signals the build stage is complete.
		AfterAll(ctx context.Context, stage *core.Stage) error

		// Watch watches for build cancellation requests.
		Watch(ctx context.Context, stage int64) (bool, error)

		// Write writes a line to the build logs
		Write(ctx context.Context, step int64, line *core.Line) error

		// Upload uploads the full logs
		Upload(ctx context.Context, step int64, r io.Reader) error

		// UploadBytes uploads the full logs
		UploadBytes(ctx context.Context, step int64, b []byte) error

		// UploadCard creates a new card
		UploadCard(ctx context.Context, step int64, input *core.CardInput) error
	}

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
)

// New returns a new Manager.
func New(
	builds core.BuildStore,
	cards core.CardStore,
	config core.ConfigService,
	converter core.ConvertService,
	events core.Pubsub,
	logs core.LogStore,
	logz core.LogStream,
	netrcs core.NetrcService,
	repos core.RepositoryStore,
	scheduler core.Scheduler,
	secrets core.SecretStore,
	globals core.GlobalSecretStore,
	status core.StatusService,
	stages core.StageStore,
	steps core.StepStore,
	system *core.System,
	users core.UserStore,
	webhook core.WebhookSender,
) BuildManager {
	return &Manager{
		Builds:    builds,
		Cards:     cards,
		Config:    config,
		Converter: converter,
		Events:    events,
		Globals:   globals,
		Logs:      logs,
		Logz:      logz,
		Netrcs:    netrcs,
		Repos:     repos,
		Scheduler: scheduler,
		Secrets:   secrets,
		Status:    status,
		Stages:    stages,
		Steps:     steps,
		System:    system,
		Users:     users,
		Webhook:   webhook,
	}
}

// Manager provides a simplified interface to the build runner so that it
// can more easily interact with the server.
type Manager struct {
	Builds    core.BuildStore
	Cards     core.CardStore
	Config    core.ConfigService
	Converter core.ConvertService
	Events    core.Pubsub
	Globals   core.GlobalSecretStore
	Logs      core.LogStore
	Logz      core.LogStream
	Netrcs    core.NetrcService
	Repos     core.RepositoryStore
	Scheduler core.Scheduler
	Secrets   core.SecretStore
	Status    core.StatusService
	Stages    core.StageStore
	Steps     core.StepStore
	System    *core.System
	Users     core.UserStore
	Webhook   core.WebhookSender
}

// Request requests the next available build stage for execution.
func (m *Manager) Request(ctx context.Context, args *Request) (*core.Stage, error) {
	logger := logrus.WithFields(
		logrus.Fields{
			"kind":    args.Kind,
			"type":    args.Type,
			"os":      args.OS,
			"arch":    args.Arch,
			"kernel":  args.Kernel,
			"variant": args.Variant,
		},
	)
	logger.Debugln("manager: request queue item")

	stage, err := m.Scheduler.Request(ctx, core.Filter{
		Kind:    args.Kind,
		Type:    args.Type,
		OS:      args.OS,
		Arch:    args.Arch,
		Kernel:  args.Kernel,
		Variant: args.Variant,
		Labels:  args.Labels,
	})
	if err != nil && ctx.Err() != nil {
		logger.Debugln("manager: context canceled")
		return nil, err
	}
	if err != nil {
		logger = logrus.WithError(err)
		logger.Warnln("manager: request queue item error")
		return nil, err
	}
	return stage, nil
}

// Accept accepts the build stage for execution. It is possible for multiple
// agents to pull the same stage from the queue. The system uses optimistic
// locking at the database-level to prevent multiple agents from executing the
// same stage.
func (m *Manager) Accept(ctx context.Context, id int64, machine string) (*core.Stage, error) {
	logger := logrus.WithFields(
		logrus.Fields{
			"stage-id": id,
			"machine":  machine,
		},
	)
	logger.Debugln("manager: accept stage")

	stage, err := m.Stages.Find(noContext, id)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find stage")
		return nil, err
	}
	if stage.Machine != "" {
		logger.Debugln("manager: stage already assigned. abort.")
		return nil, db.ErrOptimisticLock
	}

	stage.Machine = machine
	stage.Status = core.StatusPending
	stage.Updated = time.Now().Unix()

	err = m.Stages.Update(noContext, stage)
	if err == db.ErrOptimisticLock {
		logger = logger.WithError(err)
		logger.Debugln("manager: stage processed by another agent")
	} else if err != nil {
		logger = logger.WithError(err)
		logger.Debugln("manager: cannot update stage")
	} else {
		logger.Debugln("manager: stage accepted")
	}
	return stage, err
}

// Details fetches build details.
func (m *Manager) Details(ctx context.Context, id int64) (*Context, error) {
	logger := logrus.WithField("step-id", id)
	logger.Debugln("manager: fetching stage details")

	stage, err := m.Stages.Find(noContext, id)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find stage")
		return nil, err
	}
	build, err := m.Builds.Find(noContext, stage.BuildID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find build")
		return nil, err
	}
	stages, err := m.Stages.List(ctx, stage.BuildID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot list stages")
		return nil, err
	}
	build.Stages = stages
	repo, err := m.Repos.Find(noContext, build.RepoID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find repository")
		return nil, err
	}
	logger = logger.WithFields(
		logrus.Fields{
			"build": build.Number,
			"repo":  repo.Slug,
		},
	)
	user, err := m.Users.Find(noContext, repo.UserID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find repository owner")
		return nil, err
	}
	config, err := m.Config.Find(noContext, &core.ConfigArgs{
		User:  user,
		Repo:  repo,
		Build: build,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find configuration")
		return nil, err
	}

	// this code is temporarily in place to detect and convert
	// the legacy yaml configuration file to the new format.
	config.Data, _ = converter.ConvertString(config.Data, converter.Metadata{
		Filename: repo.Config,
		URL:      repo.Link,
		Ref:      build.Ref,
	})

	config, err = m.Converter.Convert(noContext, &core.ConvertArgs{
		Build:  build,
		Config: config,
		Repo:   repo,
		User:   user,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot convert configuration")
		return nil, err
	}
	var secrets []*core.Secret
	tmpSecrets, err := m.Secrets.List(noContext, repo.ID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot list secrets")
		return nil, err
	}
	tmpGlobalSecrets, err := m.Globals.List(noContext, repo.Namespace)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot list global secrets")
		return nil, err
	}
	// TODO(bradrydzewski) can we delegate filtering
	// secrets to the agent? If not, we should add
	// unit tests.
	for _, secret := range tmpSecrets {
		if secret.PullRequest == false &&
			build.Event == core.EventPullRequest {
			continue
		}
		secrets = append(secrets, secret)
	}
	for _, secret := range tmpGlobalSecrets {
		if secret.PullRequest == false &&
			build.Event == core.EventPullRequest {
			continue
		}
		secrets = append(secrets, secret)
	}
	return &Context{
		Repo:    repo,
		Build:   build,
		Stage:   stage,
		Secrets: secrets,
		System:  m.System,
		Config:  &core.File{Data: []byte(config.Data)},
	}, nil
}

// Before signals the build step is about to start.
func (m *Manager) Before(ctx context.Context, step *core.Step) error {
	logger := logrus.WithFields(
		logrus.Fields{
			"step.status": step.Status,
			"step.name":   step.Name,
			"step.id":     step.ID,
		},
	)
	logger.Debugln("manager: updating step status")

	err := m.Logz.Create(noContext, step.ID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot create log stream")
		return err
	}
	updater := &updater{
		Builds:  m.Builds,
		Events:  m.Events,
		Repos:   m.Repos,
		Steps:   m.Steps,
		Stages:  m.Stages,
		Webhook: m.Webhook,
	}
	return updater.do(ctx, step)
}

// After signals the build step is complete.
func (m *Manager) After(ctx context.Context, step *core.Step) error {
	logger := logrus.WithFields(
		logrus.Fields{
			"step.status": step.Status,
			"step.name":   step.Name,
			"step.id":     step.ID,
		},
	)
	logger.Debugln("manager: updating step status")

	var errs error
	updater := &updater{
		Builds:  m.Builds,
		Events:  m.Events,
		Repos:   m.Repos,
		Steps:   m.Steps,
		Stages:  m.Stages,
		Webhook: m.Webhook,
	}

	if err := updater.do(ctx, step); err != nil {
		errs = multierror.Append(errs, err)
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot update step")
	}

	if err := m.Logz.Delete(noContext, step.ID); err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot teardown log stream")
	}
	return errs
}

// BeforeAll signals the build stage is about to start.
func (m *Manager) BeforeAll(ctx context.Context, stage *core.Stage) error {
	s := &setup{
		Builds: m.Builds,
		Events: m.Events,
		Repos:  m.Repos,
		Steps:  m.Steps,
		Stages: m.Stages,
		Status: m.Status,
		Users:  m.Users,
	}
	return s.do(ctx, stage)
}

// AfterAll signals the build stage is complete.
func (m *Manager) AfterAll(ctx context.Context, stage *core.Stage) error {
	t := &teardown{
		Builds:    m.Builds,
		Events:    m.Events,
		Logs:      m.Logz,
		Repos:     m.Repos,
		Scheduler: m.Scheduler,
		Steps:     m.Steps,
		Stages:    m.Stages,
		Status:    m.Status,
		Users:     m.Users,
		Webhook:   m.Webhook,
	}
	return t.do(ctx, stage)
}

// Netrc returns netrc file with a valid, non-expired token
// that can be used to clone the repository.
func (m *Manager) Netrc(ctx context.Context, id int64) (*core.Netrc, error) {
	logger := logrus.WithField("repo.id", id)

	repo, err := m.Repos.Find(ctx, id)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find repository")
		return nil, err
	}

	user, err := m.Users.Find(ctx, repo.UserID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("manager: cannot find repository owner")
		return nil, err
	}

	netrc, err := m.Netrcs.Create(ctx, user, repo)
	if err != nil {
		logger = logger.WithError(err)
		logger = logger.WithField("repo.name", repo.Slug)
		logger.Warnln("manager: cannot generate netrc")
	}
	return netrc, err
}

// Watch watches for build cancellation requests.
func (m *Manager) Watch(ctx context.Context, id int64) (bool, error) {
	ok, err := m.Scheduler.Cancelled(ctx, id)
	// we expect a context cancel error here which
	// indicates a polling timeout. The subscribing
	// client should look for the context cancel error
	// and resume polling.
	if err != nil {
		return ok, err
	}

	// // TODO (bradrydzewski) we should be able to return
	// // immediately if Cancelled returns true. This requires
	// // some more testing but would avoid the extra database
	// // call.
	// if ok {
	// 	return ok, err
	// }

	// if no error is returned we should check
	// the database to see if the build is complete. If
	// complete, return true.
	build, err := m.Builds.Find(ctx, id)
	if err != nil {
		logger := logrus.WithError(err)
		logger = logger.WithField("build-id", id)
		logger.Warnln("manager: cannot find build")
		return ok, err
	}
	return build.IsDone(), nil
}

// Write writes a line to the build logs.
func (m *Manager) Write(ctx context.Context, step int64, line *core.Line) error {
	err := m.Logz.Write(ctx, step, line)
	if err != nil {
		logger := logrus.WithError(err)
		logger = logger.WithField("step-id", step)
		logger.Warnln("manager: cannot write to log stream")
	}
	return err
}

// Upload uploads the full logs.
func (m *Manager) Upload(ctx context.Context, step int64, r io.Reader) error {
	err := m.Logs.Create(ctx, step, r)
	if err != nil {
		logger := logrus.WithError(err)
		logger = logger.WithField("step-id", step)
		logger.Warnln("manager: cannot upload complete logs")
	}
	return err
}

// UploadBytes uploads the full logs.
func (m *Manager) UploadBytes(ctx context.Context, step int64, data []byte) error {
	buf := bytes.NewBuffer(data)
	err := m.Logs.Create(ctx, step, buf)
	if err != nil {
		logger := logrus.WithError(err)
		logger = logger.WithField("step-id", step)
		logger.Warnln("manager: cannot upload complete logs")
	}
	return err
}

// UploadCard creates card for step.
func (m *Manager) UploadCard(ctx context.Context, stepId int64, input *core.CardInput) error {
	data := ioutil.NopCloser(
		bytes.NewBuffer(input.Data),
	)
	err := m.Cards.Create(ctx, stepId, data)
	if err != nil {
		logger := logrus.WithError(err)
		logger.Warnln("manager: cannot create card")
	}
	return nil
}
