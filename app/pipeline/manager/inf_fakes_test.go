// Copyright 2026 Harness, Inc.
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
	"io"
	"sync"

	pipelineevents "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var errInfBoom = errors.New("boom")

type infExecutionStore struct {
	store.ExecutionStore

	findFn   func(id int64) (*types.Execution, error)
	updateFn func(e *types.Execution) error
}

func (s *infExecutionStore) Find(_ context.Context, id int64) (*types.Execution, error) {
	return s.findFn(id)
}

func (s *infExecutionStore) Update(_ context.Context, e *types.Execution) error {
	if s.updateFn == nil {
		return nil
	}
	return s.updateFn(e)
}

type infStageStore struct {
	store.StageStore

	findFn           func(id int64) (*types.Stage, error)
	updateFn         func(s *types.Stage) error
	listFn           func(executionID int64) ([]*types.Stage, error)
	listWithStepsFn  func(executionID int64) ([]*types.Stage, error)
	updated          []*types.Stage
	listWithStepsHit int
}

func (s *infStageStore) Find(_ context.Context, id int64) (*types.Stage, error) {
	return s.findFn(id)
}

func (s *infStageStore) Update(_ context.Context, st *types.Stage) error {
	s.updated = append(s.updated, st)
	if s.updateFn == nil {
		return nil
	}
	return s.updateFn(st)
}

func (s *infStageStore) List(_ context.Context, executionID int64) ([]*types.Stage, error) {
	return s.listFn(executionID)
}

func (s *infStageStore) ListWithSteps(_ context.Context, executionID int64) ([]*types.Stage, error) {
	s.listWithStepsHit++
	return s.listWithStepsFn(executionID)
}

type infStepStore struct {
	store.StepStore

	createFn func(s *types.Step) error
	updateFn func(s *types.Step) error
	created  []*types.Step
	updated  []*types.Step
}

func (s *infStepStore) Create(_ context.Context, st *types.Step) error {
	s.created = append(s.created, st)
	if s.createFn == nil {
		return nil
	}
	return s.createFn(st)
}

func (s *infStepStore) Update(_ context.Context, st *types.Step) error {
	s.updated = append(s.updated, st)
	if s.updateFn == nil {
		return nil
	}
	return s.updateFn(st)
}

type infRepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (s *infRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	if s.err != nil {
		return nil, s.err
	}
	cp := *s.repo
	return &cp, nil
}

type infPipelineStore struct {
	store.PipelineStore

	pipeline *types.Pipeline
	err      error
}

func (s *infPipelineStore) Find(context.Context, int64) (*types.Pipeline, error) {
	return s.pipeline, s.err
}

type infCheckStore struct {
	store.CheckStore

	upserted []*types.Check
	err      error
}

func (s *infCheckStore) Upsert(_ context.Context, c *types.Check) error {
	s.upserted = append(s.upserted, c)
	return s.err
}

type infSecretStore struct {
	store.SecretStore

	secrets []*types.Secret
	err     error
}

func (s *infSecretStore) ListAll(context.Context, int64) ([]*types.Secret, error) {
	return s.secrets, s.err
}

type infLogStore struct {
	store.LogStore

	mx      sync.Mutex
	uploads map[int64]string
	err     error
}

func (s *infLogStore) Create(_ context.Context, stepID int64, r io.Reader) error {
	if s.err != nil {
		return s.err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.uploads == nil {
		s.uploads = map[int64]string{}
	}
	s.uploads[stepID] = string(data)
	return nil
}

type infLogStream struct {
	livelog.LogStream

	createErr error
	deleteErr error
	writeErr  error
	created   []int64
	deleted   []int64
	lines     []*livelog.Line
}

func (s *infLogStream) Create(_ context.Context, id int64) error {
	s.created = append(s.created, id)
	return s.createErr
}

func (s *infLogStream) Delete(_ context.Context, id int64) error {
	s.deleted = append(s.deleted, id)
	return s.deleteErr
}

func (s *infLogStream) Write(_ context.Context, _ int64, l *livelog.Line) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.lines = append(s.lines, l)
	return nil
}

type infScheduler struct {
	scheduler.Scheduler

	requestFn   func(f scheduler.Filter) (*types.Stage, error)
	cancelledFn func(id int64) (bool, error)
	scheduleErr error
	scheduled   []*types.Stage
}

func (s *infScheduler) Request(_ context.Context, f scheduler.Filter) (*types.Stage, error) {
	return s.requestFn(f)
}

func (s *infScheduler) Cancelled(_ context.Context, id int64) (bool, error) {
	return s.cancelledFn(id)
}

func (s *infScheduler) Schedule(_ context.Context, st *types.Stage) error {
	s.scheduled = append(s.scheduled, st)
	return s.scheduleErr
}

type infFileService struct {
	data string
	err  error
}

func (s *infFileService) Get(context.Context, *types.Repository, string, string) (*file.File, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &file.File{Data: []byte(s.data)}, nil
}

type infConverter struct {
	out *file.File
	err error
}

func (c *infConverter) Convert(_ context.Context, args *converter.ConvertArgs) (*file.File, error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.out != nil {
		return c.out, nil
	}
	return args.File, nil
}

type infPublicAccess struct {
	publicaccess.Service

	public bool
	err    error
}

func (p *infPublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return p.public, p.err
}

type infURLProvider struct {
	url.Provider
}

func (infURLProvider) GenerateContainerGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.invalid/" + repoPath + ".git"
}

func (infURLProvider) GetAPIProto(context.Context) string { return "https" }

func (infURLProvider) GetAPIHostname(context.Context) string { return "api.example.invalid" }

// infReporter returns a real pipeline events reporter backed by an in-memory events system.
func infReporter() pipelineevents.Reporter {
	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "inf", MaxStreamLength: 10},
		nil, events.NewNoopCollector())
	if err != nil {
		panic(err)
	}
	rep, err := pipelineevents.NewReporter(sys)
	if err != nil {
		panic(err)
	}
	return *rep
}
