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

package ide

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*SSH)(nil)

// SSHConfig defines SSH IDE specific configuration.
type SSHConfig struct {
	Port int
}

// SSH implements the IDE interface for direct SSH access to the container.
type SSH struct {
	config *SSHConfig
}

// NewSSHService creates a new SSH IDE service.
func NewSSHService(config *SSHConfig) *SSH {
	return &SSH{config: config}
}

// Setup installs the SSH server inside the container.
func (s *SSH) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container...")
	err := setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup SSH server: %w", err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")
	gitspaceLogger.Info("Successfully set up SSH inside container")
	return nil
}

// Run starts the SSH server inside the container.
func (s *SSH) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Starting ssh-server...")
	err := runSSHServer(ctx, exec, s.config.Port, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run SSH server: %w", err)
	}
	gitspaceLogger.Info("Successfully started ssh-server")
	return nil
}

// Port returns the port on which the ssh-server is listening.
func (s *SSH) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     s.config.Port,
		Protocol: enum.CommunicationProtocolSSH,
	}
}

// Type returns the IDE type this service represents.
func (s *SSH) Type() enum.IDEType {
	return enum.IDETypeSSH
}

// GenerateURL returns a direct SSH command that the user can run to connect.
// SSH IDE provides direct SSH access to the container for command-line usage.
//
//	eg: ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 2222 \
//	            vscode@zealous-pyro-e714-qw2u6x-h4w0zj.us-west-ga.gitspace.qa.harness.io
func (s *SSH) GenerateURL(absoluteRepoPath, host, port, user string) string { //nolint:revive // match interface
	// absoluteRepoPath is intentionally unused for SSH command.
	return fmt.Sprintf(
		"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %s %s@%s",
		port,
		user,
		host,
	)
}
