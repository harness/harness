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
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const RootUser = "root"
const ErrMsgTCP = "unable to upgrade to tcp, received 200"

type Exec struct {
	ContainerName  string
	DockerClient   *client.Client
	HomeDir        string
	UserIdentifier string
	AccessKey      string
	AccessType     enum.GitspaceAccessType
}

type execResult struct {
	StdOut   io.Reader
	StdErr   io.Reader
	ExitCode int
}

func (e *Exec) ExecuteCommand(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
	workingDir string,
	outputCh chan []byte, // channel to stream output as []byte
) error {
	user := e.UserIdentifier
	if root {
		user = RootUser
	}

	cmd := []string{"/bin/sh", "-c", command}
	execConfig := container.ExecOptions{
		User:         user,
		AttachStdout: !detach,
		AttachStderr: !detach,
		Cmd:          cmd,
		Detach:       detach,
		WorkingDir:   workingDir,
	}

	// Create exec instance for the container
	containerExecCreate, err := e.DockerClient.ContainerExecCreate(ctx, e.ContainerName, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create docker exec for container %s: %w", e.ContainerName, err)
	}

	// Attach and inspect exec session to get the output
	inspectExec, err := e.attachAndInspectExec(ctx, containerExecCreate.ID, detach)
	if err != nil && !strings.Contains(err.Error(), ErrMsgTCP) {
		return fmt.Errorf("failed to start docker exec for container %s: %w", e.ContainerName, err)
	}
	// If in detach mode, exit early as the command will run in the background
	if detach {
		close(outputCh)
		return nil
	}

	// Wait for the exit code after the command completes
	if inspectExec != nil && inspectExec.ExitCode != 0 {
		return fmt.Errorf("error during command execution in container %s. exit code %d",
			e.ContainerName, inspectExec.ExitCode)
	}

	e.streamResponse(inspectExec, outputCh)
	return nil
}

func (e *Exec) ExecuteCommandInHomeDirectory(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
	outputCh chan []byte, // channel to stream output as []byte
) error {
	return e.ExecuteCommand(ctx, command, root, detach, e.HomeDir, outputCh)
}

func (e *Exec) attachAndInspectExec(ctx context.Context, id string, detach bool) (*execResult, error) {
	resp, attachErr := e.DockerClient.ContainerExecAttach(ctx, id, container.ExecStartOptions{Detach: detach})
	if attachErr != nil {
		return nil, fmt.Errorf("failed to attach to exec session: %w", attachErr)
	}

	// If in detach mode, we just need to close the connection, not process output
	if detach {
		// No need to process output in detach mode, so we simply close the connection
		resp.Close()
		return nil, nil //nolint:nilnil
	}

	// Create pipes for stdout and stderr
	stdoutPipe, stdoutWriter := io.Pipe()
	stderrPipe, stderrWriter := io.Pipe()

	go e.copyOutput(resp.Reader, stdoutWriter, stderrWriter)

	// Return the output streams and the response
	return &execResult{
		StdOut: stdoutPipe, // Pipe for stdout
		StdErr: stderrPipe, // Pipe for stderr
	}, nil
}

func (e *Exec) streamResponse(resp *execResult, outputCh chan []byte) {
	// Stream the output asynchronously if not in detach mode
	go func() {
		if resp != nil {
			var wg sync.WaitGroup

			// Handle stdout as a streaming reader
			if resp.StdOut != nil {
				wg.Add(1)
				go e.streamStdOut(resp.StdOut, outputCh, &wg)
			}
			// Handle stderr as a streaming reader
			if resp.StdErr != nil {
				wg.Add(1)
				go e.streamStdErr(resp.StdErr, outputCh, &wg)
			}
			// Wait for all readers to finish before closing the channel
			wg.Wait()
			// Close the output channel after all output has been processed
			close(outputCh)
		}
	}()
}

// copyOutput copies the output from the exec response to the pipes, and is blocking.
func (e *Exec) copyOutput(reader io.Reader, stdoutWriter, stderrWriter io.WriteCloser) {
	_, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
	if err != nil {
		log.Printf("Error copying output: %v", err)
	}
	stdoutWriter.Close()
	stderrWriter.Close()
}

// streamStdOut reads from the stdout pipe and sends each line to the output channel.
func (e *Exec) streamStdOut(stdout io.Reader, outputCh chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	stdoutReader := bufio.NewScanner(stdout)
	for stdoutReader.Scan() {
		outputCh <- stdoutReader.Bytes()
	}
	if err := stdoutReader.Err(); err != nil {
		log.Println("Error reading stdout:", err)
	}
}

// streamStdErr reads from the stderr pipe and sends each line to the output channel.
func (e *Exec) streamStdErr(stderr io.Reader, outputCh chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	stderrReader := bufio.NewScanner(stderr)
	for stderrReader.Scan() {
		outputCh <- []byte("ERR> " + stderrReader.Text())
	}
	if err := stderrReader.Err(); err != nil {
		log.Println("Error reading stderr:", err)
	}
}
