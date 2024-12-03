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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/harness/gitness/types/enum"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
)

const RootUser = "root"
const ErrMsgTCP = "unable to upgrade to tcp, received 200"
const LoggerErrorPrefix = "ERR>> "
const ChannelExitStatus = "DEVCONTAINER_EXIT_STATUS"

type Exec struct {
	ContainerName     string
	DockerClient      *client.Client
	DefaultWorkingDir string
	RemoteUser        string
	AccessKey         string
	AccessType        enum.GitspaceAccessType
}

type execResult struct {
	ExecID string
	StdOut io.Reader
	StdErr io.Reader
}

func (e *Exec) ExecuteCommand(
	ctx context.Context,
	command string,
	root bool,
	workingDir string,
) (string, error) {
	containerExecCreate, err := e.createExecution(ctx, command, root, workingDir, false)
	if err != nil {
		return "", fmt.Errorf("failed to create exec instance: %w", err)
	}

	resp, err := e.DockerClient.ContainerExecAttach(
		ctx, containerExecCreate.ID, container.ExecStartOptions{Detach: false})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec session: %w", err)
	}
	defer resp.Close()

	// Prepare buffers for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer

	// Use stdcopy to demultiplex output
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, resp.Reader)
	if err != nil {
		return "", fmt.Errorf("error during stdcopy: %w", err)
	}
	inspect, err := e.DockerClient.ContainerExecInspect(ctx, containerExecCreate.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec session: %w", err)
	}
	// Handle non-zero exit codes
	if inspect.ExitCode != 0 {
		return fmt.Sprintf(
			"STDOUT:\n%s\nSTDERR:\n%s", stdoutBuf.String(), stderrBuf.String(),
		), fmt.Errorf("command exited with non-zero status: %d", inspect.ExitCode)
	}
	return stdoutBuf.String(), nil
}

func (e *Exec) createExecution(
	ctx context.Context,
	command string,
	root bool,
	workingDir string,
	detach bool,
) (*dockerTypes.IDResponse, error) {
	user := e.RemoteUser
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
	log.Debug().Msgf("Creating execution for container %s", e.ContainerName)
	containerExecCreate, err := e.DockerClient.ContainerExecCreate(ctx, e.ContainerName, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker exec for container %s: %w", e.ContainerName, err)
	}
	return &containerExecCreate, nil
}

func (e *Exec) executeCmdAsyncStream(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
	workingDir string,
	outputCh chan []byte, // channel to stream output as []byte
) error {
	containerExecCreate, err := e.createExecution(ctx, command, root, workingDir, detach)
	if err != nil {
		return err
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
	e.streamResponse(inspectExec, outputCh)
	return nil
}

func (e *Exec) ExecuteCmdInHomeDirectoryAsyncStream(
	ctx context.Context,
	command string,
	root bool,
	detach bool,
	outputCh chan []byte, // channel to stream output as []byte
) error {
	return e.executeCmdAsyncStream(ctx, command, root, detach, e.DefaultWorkingDir, outputCh)
}

func (e *Exec) attachAndInspectExec(ctx context.Context, id string, detach bool) (*execResult, error) {
	resp, attachErr := e.DockerClient.ContainerExecAttach(ctx, id, container.ExecStartOptions{Detach: detach})
	if attachErr != nil {
		return nil, fmt.Errorf("failed to attach to exec session: %w", attachErr)
	}

	// If in detach mode, we just need to close the connection, not process output
	if detach {
		resp.Close()
		return nil, nil //nolint:nilnil
	}

	// Create pipes for stdout and stderr
	stdoutPipe, stdoutWriter := io.Pipe()
	stderrPipe, stderrWriter := io.Pipe()

	go e.copyOutput(resp, stdoutWriter, stderrWriter)

	// Return the output streams and the response
	return &execResult{
		ExecID: id,
		StdOut: stdoutPipe, // Pipe for stdout
		StdErr: stderrPipe, // Pipe for stderr
	}, nil
}

func (e *Exec) streamResponse(resp *execResult, outputCh chan []byte) {
	// Stream the output asynchronously if not in detach mode
	go func() {
		defer close(outputCh)
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

			// Now that streaming is finished, inspect the exit status
			log.Debug().Msgf("Inspecting container for status: %s", resp.ExecID)
			inspect, err := e.DockerClient.ContainerExecInspect(context.Background(), resp.ExecID)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to inspect exec session: %s", err.Error())
				return
			}

			// Send the exit status as a final message
			exitStatusMsg := fmt.Sprintf(ChannelExitStatus+"%d", inspect.ExitCode)
			outputCh <- []byte(exitStatusMsg)
		}
	}()
}

func (e *Exec) copyOutput(response dockerTypes.HijackedResponse, stdoutWriter, stderrWriter io.WriteCloser) {
	defer func() {
		if err := stdoutWriter.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing stdoutWriter")
		}
		if err := stderrWriter.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing stderrWriter")
		}
		response.Close()
	}()
	_, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, response.Reader)
	if err != nil {
		log.Error().Err(err).Msg("Error in stdcopy.StdCopy " + err.Error())
	}
}

// streamStdOut reads from the stdout pipe and sends each line to the output channel.
func (e *Exec) streamStdOut(stdout io.Reader, outputCh chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	stdoutReader := bufio.NewScanner(stdout)
	for stdoutReader.Scan() {
		select {
		case <-context.Background().Done():
			log.Info().Msg("Context canceled, stopping stdout streaming")
			return
		default:
			outputCh <- stdoutReader.Bytes()
		}
	}
	if err := stdoutReader.Err(); err != nil {
		log.Error().Err(err).Msg("Error reading stdout " + err.Error())
	}
}

// streamStdErr reads from the stderr pipe and sends each line to the output channel.
func (e *Exec) streamStdErr(stderr io.Reader, outputCh chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	stderrReader := bufio.NewScanner(stderr)
	for stderrReader.Scan() {
		select {
		case <-context.Background().Done():
			log.Info().Msg("Context canceled, stopping stderr streaming")
			return
		default:
			outputCh <- []byte(LoggerErrorPrefix + stderrReader.Text())
		}
	}
	if err := stderrReader.Err(); err != nil {
		log.Error().Err(err).Msg("Error reading stderr " + err.Error())
	}
}
