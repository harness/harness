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
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/drone-go/drone"
	"github.com/drone/runner-go/client"
	"github.com/stretchr/testify/require"
)

// infExecutionManager records calls made by the embedded client.
type infExecutionManager struct {
	requestArgs *Request
	requestErr  error
	stage       *types.Stage
	acceptErr   error
	details     *ExecutionContext
	detailsErr  error
	calls       []string
	stageArg    *types.Stage
	stepArg     *types.Step
	beforeErr   error
	writeErr    error
	lines       []*livelog.Line
	uploaded    string
	uploadErr   error
	watchResult bool
	watchErr    error
}

func (m *infExecutionManager) Request(_ context.Context, args *Request) (*types.Stage, error) {
	m.requestArgs = args
	return m.stage, m.requestErr
}

func (m *infExecutionManager) Watch(context.Context, int64) (bool, error) {
	return m.watchResult, m.watchErr
}

func (m *infExecutionManager) Accept(_ context.Context, _ int64, machine string) (*types.Stage, error) {
	if m.acceptErr != nil {
		return nil, m.acceptErr
	}
	cp := *m.stage
	cp.Machine = machine
	return &cp, nil
}

func (m *infExecutionManager) Write(_ context.Context, _ int64, line *livelog.Line) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.lines = append(m.lines, line)
	return nil
}

func (m *infExecutionManager) Details(context.Context, int64) (*ExecutionContext, error) {
	return m.details, m.detailsErr
}

func (m *infExecutionManager) UploadLogs(_ context.Context, _ int64, r io.Reader) error {
	data, _ := io.ReadAll(r)
	m.uploaded = string(data)
	return m.uploadErr
}

func (m *infExecutionManager) BeforeStep(_ context.Context, s *types.Step) error {
	m.calls = append(m.calls, "BeforeStep")
	m.stepArg = s
	return m.beforeErr
}

func (m *infExecutionManager) AfterStep(_ context.Context, s *types.Step) error {
	m.calls = append(m.calls, "AfterStep")
	m.stepArg = s
	return m.beforeErr
}

func (m *infExecutionManager) BeforeStage(_ context.Context, s *types.Stage) error {
	m.calls = append(m.calls, "BeforeStage")
	m.stageArg = s
	return m.beforeErr
}

func (m *infExecutionManager) AfterStage(_ context.Context, s *types.Stage) error {
	m.calls = append(m.calls, "AfterStage")
	m.stageArg = s
	return m.beforeErr
}

func TestInfEmbeddedClient_NoopMethods(t *testing.T) {
	c := NewEmbeddedClient(&infExecutionManager{}, infURLProvider{}, &types.Config{})
	ctx := context.Background()
	require.NoError(t, c.Join(ctx, "m"))
	require.NoError(t, c.Leave(ctx, "m"))
	require.NoError(t, c.Ping(ctx, "m"))
	require.NoError(t, c.UploadCard(ctx, 1, &drone.CardInput{}))
}

func TestInfEmbeddedClient_RequestAndAccept(t *testing.T) {
	ctx := context.Background()
	m := &infExecutionManager{stage: &types.Stage{ID: 4, Name: "build", Started: 5000}}
	c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})

	stage, err := c.Request(ctx, &client.Filter{
		Kind: "pipeline", Type: "docker", OS: "linux", Arch: "arm64", Variant: "v8", Kernel: "5.10",
		Labels: map[string]string{"x": "y"},
	})
	require.NoError(t, err)
	require.EqualValues(t, 4, stage.ID)
	require.EqualValues(t, 5, stage.Started, "milliseconds are converted to seconds")
	require.Equal(t, &Request{
		Kind: "pipeline", Type: "docker", OS: "linux", Arch: "arm64", Variant: "v8", Kernel: "5.10",
		Labels: map[string]string{"x": "y"},
	}, m.requestArgs)

	m.requestErr = errInfBoom
	_, err = c.Request(ctx, &client.Filter{})
	require.ErrorIs(t, err, errInfBoom)

	accepted := &drone.Stage{ID: 4, Machine: "box"}
	require.NoError(t, c.Accept(ctx, accepted))
	require.Equal(t, "box", accepted.Machine)
	require.Equal(t, "build", accepted.Name, "stage is refreshed from the manager's answer")

	m.acceptErr = errInfBoom
	require.ErrorIs(t, c.Accept(ctx, &drone.Stage{}), errInfBoom)
}

func TestInfEmbeddedClient_Detail(t *testing.T) {
	ctx := context.Background()
	m := &infExecutionManager{details: &ExecutionContext{
		Repo:         &types.Repository{ID: 1, Identifier: "repo", Path: "space/repo"},
		RepoIsPublic: true,
		Execution:    &types.Execution{ID: 2, Event: enum.TriggerEventPush},
		Stage:        &types.Stage{ID: 3},
		Secrets:      []*types.Secret{{Identifier: "token", Data: "value"}},
		Config:       &file.File{Data: []byte("kind: pipeline")},
		Netrc:        &Netrc{Machine: "git", Login: "svc", Password: "pw"},
	}}
	c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})

	got, err := c.Detail(ctx, &drone.Stage{ID: 3})
	require.NoError(t, err)
	require.EqualValues(t, 2, got.Build.ID)
	require.False(t, got.Repo.Private)
	require.True(t, got.Repo.Trusted)
	require.EqualValues(t, 3, got.Stage.ID)
	require.Equal(t, "token", got.Secrets[0].Name)
	require.Equal(t, "kind: pipeline", string(got.Config.Data))
	require.Equal(t, "pw", got.Netrc.Password)
	require.Equal(t, "https", got.System.Proto)
	require.Equal(t, "api.example.invalid", got.System.Host)

	m.detailsErr = errInfBoom
	_, err = c.Detail(ctx, &drone.Stage{})
	require.ErrorIs(t, err, errInfBoom)
}

func TestInfEmbeddedClient_UpdateDispatchesOnStatus(t *testing.T) {
	ctx := context.Background()

	for status, want := range map[string]string{
		"pending": "BeforeStage", "running": "BeforeStage",
		"success": "AfterStage", "failure": "AfterStage", "killed": "AfterStage",
	} {
		m := &infExecutionManager{}
		c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})
		stage := &drone.Stage{ID: 1, Status: status}
		require.NoError(t, c.Update(ctx, stage))
		require.Equal(t, []string{want}, m.calls, status)
		require.Equal(t, status, stage.Status)
	}

	m := &infExecutionManager{beforeErr: errInfBoom}
	c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})
	require.ErrorIs(t, c.Update(ctx, &drone.Stage{Status: "running"}), errInfBoom)
}

func TestInfEmbeddedClient_UpdateStepDispatchesOnStatus(t *testing.T) {
	ctx := context.Background()

	for status, want := range map[string]string{
		"pending": "BeforeStep", "running": "BeforeStep", "success": "AfterStep", "error": "AfterStep",
	} {
		m := &infExecutionManager{}
		c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})
		step := &drone.Step{ID: 1, Status: status, Started: 10}
		require.NoError(t, c.UpdateStep(ctx, step))
		require.Equal(t, []string{want}, m.calls, status)
		require.EqualValues(t, 10, step.Started)
	}

	m := &infExecutionManager{beforeErr: errInfBoom}
	c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})
	require.ErrorIs(t, c.UpdateStep(ctx, &drone.Step{Status: "success"}), errInfBoom)
}

func TestInfEmbeddedClient_WatchAndLogs(t *testing.T) {
	ctx := context.Background()
	m := &infExecutionManager{watchResult: true}
	c := NewEmbeddedClient(m, infURLProvider{}, &types.Config{})

	done, err := c.Watch(ctx, 1)
	require.NoError(t, err)
	require.True(t, done)
	m.watchErr = errInfBoom
	_, err = c.Watch(ctx, 1)
	require.ErrorIs(t, err, errInfBoom)

	lines := []*drone.Line{{Number: 1, Message: "a", Timestamp: 2}, {Number: 2, Message: "b"}}
	require.NoError(t, c.Batch(ctx, 9, lines))
	require.Len(t, m.lines, 2)
	require.Equal(t, livelog.Line{Number: 1, Message: "a", Timestamp: 2}, *m.lines[0])
	m.writeErr = errInfBoom
	require.ErrorIs(t, c.Batch(ctx, 9, lines), errInfBoom)

	require.NoError(t, c.Upload(ctx, 9, lines))
	var uploaded []livelog.Line
	require.NoError(t, json.Unmarshal([]byte(m.uploaded), &uploaded))
	require.Equal(t, []livelog.Line{{Number: 1, Message: "a", Timestamp: 2}, {Number: 2, Message: "b"}}, uploaded)
	m.uploadErr = errors.New("upload failed")
	require.EqualError(t, c.Upload(ctx, 9, lines), "upload failed")
}
