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

package container_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"

	"github.com/docker/docker/client"
)

// recordingLogger is a no-op GitspaceLogger that records the messages passed to Error, so a test
// can assert what was logged without a real log stream. It is written from multiple goroutines
// (commands run in parallel), so access is guarded by a mutex.
type recordingLogger struct {
	mu     sync.Mutex
	errors []string
}

func (l *recordingLogger) Info(string)  {}
func (l *recordingLogger) Debug(string) {}
func (l *recordingLogger) Warn(string)  {}
func (l *recordingLogger) Error(msg string, _ error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

func (l *recordingLogger) errorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.errors)
}

// newUnreachableExec returns an Exec whose Docker client points at a closed port, so every command
// it runs fails at the dial. That lets the test exercise the failure path without a Docker daemon.
func newUnreachableExec(t *testing.T) devcontainer.Exec {
	t.Helper()
	dockerClient, err := client.NewClientWithOpts(client.WithHost("tcp://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("failed to build docker client: %v", err)
	}
	t.Cleanup(func() { _ = dockerClient.Close() })
	return devcontainer.Exec{
		ContainerName: "gitspace-test-nonexistent",
		DockerClient:  dockerClient,
		RemoteUser:    "harness",
	}
}

func TestExecuteLifecycleCommands_NoCommandsReturnsNil(t *testing.T) {
	err := container.ExecuteLifecycleCommands(
		context.Background(),
		newUnreachableExec(t),
		"/workspace",
		&recordingLogger{},
		nil,
		container.PostCreateAction,
	)
	if err != nil {
		t.Fatalf("expected nil error for no commands, got %v", err)
	}
}

// A failing lifecycle command must surface as a non-nil error so the step framework's
// StopOnFailure can abort setup. Before the fix the function logged the error and returned nil,
// so setup was reported successful on a broken environment.
func TestExecuteLifecycleCommands_FailingCommandPropagates(t *testing.T) {
	logger := &recordingLogger{}
	err := container.ExecuteLifecycleCommands(
		context.Background(),
		newUnreachableExec(t),
		"/workspace",
		logger,
		[]string{"echo hello"},
		container.PostCreateAction,
	)
	if err == nil {
		t.Fatal("expected a non-nil error when a lifecycle command fails, got nil")
	}
	if logger.errorCount() == 0 {
		t.Fatal("expected the failure to be logged as well as returned")
	}
}

// When several commands fail, every failure must be represented in the returned error, not just
// the first, so none is silently dropped.
func TestExecuteLifecycleCommands_MultipleFailuresAllReported(t *testing.T) {
	err := container.ExecuteLifecycleCommands(
		context.Background(),
		newUnreachableExec(t),
		"/workspace",
		&recordingLogger{},
		[]string{"first-command", "second-command"},
		container.PostStartAction,
	)
	if err == nil {
		t.Fatal("expected a non-nil error when lifecycle commands fail, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"first-command", "second-command"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected joined error to mention %q, got: %s", want, msg)
		}
	}
}
