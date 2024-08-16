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

package devcontainer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const RootUser = "root"

type Exec struct {
	ContainerName  string
	DockerClient   *client.Client
	HomeDir        string
	UserIdentifier string
	AccessKey      string
	AccessType     enum.GitspaceAccessType
}

type execResult struct {
	StdOut   []byte
	StdErr   []byte
	ExitCode int
}

func (e *Exec) ExecuteCommand(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
	workingDir string,
) ([]byte, error) {
	user := e.UserIdentifier
	if root {
		user = RootUser
	}

	cmd := []string{"/bin/sh", "-c", command}

	execConfig := container.ExecOptions{
		User:         user,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		Detach:       detach,
		WorkingDir:   workingDir,
	}

	execID, err := e.DockerClient.ContainerExecCreate(ctx, e.ContainerName, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker exec for container %s: %w", e.ContainerName, err)
	}

	resp, err := e.attachAndInspectExec(ctx, execID.ID, detach)
	if err != nil && err.Error() != "unable to upgrade to tcp, received 200" {
		return nil, fmt.Errorf("failed to start docker exec for container %s: %w", e.ContainerName, err)
	}

	if resp != nil && resp.ExitCode != 0 {
		var errLog string
		if resp.StdErr != nil {
			errLog = string(resp.StdErr)
		}

		return nil, fmt.Errorf("error during command execution in container %s. exit code %d. log: %s",
			e.ContainerName, resp.ExitCode, errLog)
	}

	var stdOutput []byte
	if resp != nil {
		stdOutput = resp.StdOut
	}

	return stdOutput, nil
}

func (e *Exec) ExecuteCommandInHomeDirectory(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
) ([]byte, error) {
	return e.ExecuteCommand(ctx, command, root, detach, e.HomeDir)
}

func (e *Exec) attachAndInspectExec(ctx context.Context, id string, detach bool) (*execResult, error) {
	resp, attachErr := e.DockerClient.ContainerExecAttach(ctx, id, container.ExecStartOptions{Detach: detach})
	if attachErr != nil {
		return nil, attachErr
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer
	copyErr := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err := stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
		copyErr <- err
	}()

	select {
	case err := <-copyErr:
		if err != nil {
			return nil, err
		}
		break

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	stdout, err := io.ReadAll(&outBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout of exec for container %s: %w", e.ContainerName, err)
	}

	stderr, err := io.ReadAll(&errBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr of exec for container %s: %w", e.ContainerName, err)
	}

	inspectRes, err := e.DockerClient.ContainerExecInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec for container %s: %w", e.ContainerName, err)
	}

	return &execResult{
		StdOut:   stdout,
		StdErr:   stderr,
		ExitCode: inspectRes.ExitCode,
	}, nil
}
