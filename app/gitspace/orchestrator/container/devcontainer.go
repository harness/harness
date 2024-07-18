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

package container

import (
	"context"
	"fmt"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Devcontainer struct {
	ContainerName string
	WorkingDir    string
	DockerClient  *client.Client
}

func (d *Devcontainer) ExecuteCommand(ctx context.Context, command string, detach bool) ([]byte, error) {
	cmd := []string{"/bin/sh", "-c", command}

	execConfig := dockerTypes.ExecConfig{
		User:         "root",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		Detach:       detach,
		WorkingDir:   d.WorkingDir,
	}

	execID, err := d.DockerClient.ContainerExecCreate(ctx, d.ContainerName, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker exec for container %s: %w", d.ContainerName, err)
	}

	execResponse, err := d.DockerClient.ContainerExecAttach(ctx, execID.ID, dockerTypes.ExecStartCheck{Detach: detach})
	if err != nil && err.Error() != "unable to upgrade to tcp, received 200" {
		return nil, fmt.Errorf("failed to start docker exec for container %s: %w", d.ContainerName, err)
	}

	if execResponse.Conn != nil {
		defer execResponse.Close()
	}

	var output []byte
	if execResponse.Reader != nil {
		output, err = io.ReadAll(execResponse.Reader)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
