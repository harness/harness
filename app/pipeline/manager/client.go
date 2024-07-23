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
	"bytes"
	"context"
	"encoding/json"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-go/drone"
	"github.com/drone/runner-go/client"
)

type embedded struct {
	config      *types.Config
	urlProvider url.Provider
	manager     ExecutionManager
}

var _ client.Client = (*embedded)(nil)

func NewEmbeddedClient(
	manager ExecutionManager,
	urlProvider url.Provider,
	config *types.Config,
) client.Client {
	return &embedded{
		config:      config,
		urlProvider: urlProvider,
		manager:     manager,
	}
}

// Join notifies the server the runner is joining the cluster.
// Since the runner is embedded, this can just return nil.
func (e *embedded) Join(_ context.Context, _ string) error {
	return nil
}

// Leave notifies the server the runner is leaving the cluster.
// Since the runner is embedded, this can just return nil.
func (e *embedded) Leave(_ context.Context, _ string) error {
	return nil
}

// Ping sends a ping message to the server to test connectivity.
// Since the runner is embedded, this can just return nil.
func (e *embedded) Ping(_ context.Context, _ string) error {
	return nil
}

// Request requests the next available build stage for execution.
func (e *embedded) Request(ctx context.Context, args *client.Filter) (*drone.Stage, error) {
	request := &Request{
		Kind:    args.Kind,
		Type:    args.Type,
		OS:      args.OS,
		Arch:    args.Arch,
		Variant: args.Variant,
		Kernel:  args.Kernel,
		Labels:  args.Labels,
	}
	stage, err := e.manager.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	return ConvertToDroneStage(stage), nil
}

// Accept accepts the build stage for execution.
func (e *embedded) Accept(ctx context.Context, s *drone.Stage) error {
	stage, err := e.manager.Accept(ctx, s.ID, s.Machine)
	if err != nil {
		return err
	}
	*s = *ConvertToDroneStage(stage)
	return err
}

// Detail gets the build stage details for execution.
func (e *embedded) Detail(ctx context.Context, stage *drone.Stage) (*client.Context, error) {
	details, err := e.manager.Details(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	return &client.Context{
		Build:   ConvertToDroneBuild(details.Execution),
		Repo:    ConvertToDroneRepo(details.Repo, details.RepoIsPublic),
		Stage:   ConvertToDroneStage(details.Stage),
		Secrets: ConvertToDroneSecrets(details.Secrets),
		Config:  ConvertToDroneFile(details.Config),
		Netrc:   ConvertToDroneNetrc(details.Netrc),
		System: &drone.System{
			Proto: e.urlProvider.GetAPIProto(ctx),
			Host:  e.urlProvider.GetAPIHostname(ctx),
		},
	}, nil
}

// Update updates the build stage.
func (e *embedded) Update(ctx context.Context, stage *drone.Stage) error {
	var err error
	convertedStage := ConvertFromDroneStage(stage)
	status := enum.ParseCIStatus(stage.Status)
	if status == enum.CIStatusPending || status == enum.CIStatusRunning {
		err = e.manager.BeforeStage(ctx, convertedStage)
	} else {
		err = e.manager.AfterStage(ctx, convertedStage)
	}
	*stage = *ConvertToDroneStage(convertedStage)
	return err
}

// UpdateStep updates the build step.
func (e *embedded) UpdateStep(ctx context.Context, step *drone.Step) error {
	var err error
	convertedStep := ConvertFromDroneStep(step)
	status := enum.ParseCIStatus(step.Status)
	if status == enum.CIStatusPending || status == enum.CIStatusRunning {
		err = e.manager.BeforeStep(ctx, convertedStep)
	} else {
		err = e.manager.AfterStep(ctx, convertedStep)
	}
	*step = *ConvertToDroneStep(convertedStep)
	return err
}

// Watch watches for build cancellation requests.
func (e *embedded) Watch(ctx context.Context, executionID int64) (bool, error) {
	return e.manager.Watch(ctx, executionID)
}

// Batch batch writes logs to the streaming logs.
func (e *embedded) Batch(ctx context.Context, step int64, lines []*drone.Line) error {
	for _, l := range lines {
		line := ConvertFromDroneLine(l)
		err := e.manager.Write(ctx, step, line)
		if err != nil {
			return err
		}
	}
	return nil
}

// Upload uploads the full logs to the server.
func (e *embedded) Upload(ctx context.Context, step int64, l []*drone.Line) error {
	var buffer bytes.Buffer
	lines := []livelog.Line{}
	for _, line := range l {
		lines = append(lines, *ConvertFromDroneLine(line))
	}
	out, err := json.Marshal(lines)
	if err != nil {
		return err
	}
	_, err = buffer.Write(out)
	if err != nil {
		return err
	}
	return e.manager.UploadLogs(ctx, step, &buffer)
}

// UploadCard uploads a card to drone server.
func (e *embedded) UploadCard(_ context.Context, _ int64, _ *drone.CardInput) error {
	// Implement UploadCard logic here
	return nil // Replace with appropriate error handling and logic
}
