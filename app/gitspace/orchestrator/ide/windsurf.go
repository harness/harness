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

var _ IDE = (*Windsurf)(nil)

// WindsurfConfig defines Windsurf IDE specific configuration.
type WindsurfConfig struct {
	Port int
}

// Windsurf implements the IDE interface for Windsurf IDE (VSCode-based).
type Windsurf struct {
	config *WindsurfConfig
}

// NewWindsurfService creates a new Windsurf IDE service.
func NewWindsurfService(config *WindsurfConfig) *Windsurf {
	return &Windsurf{config: config}
}

// Setup installs the SSH server inside the container.
func (w *Windsurf) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container...")
	err := setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", enum.IDETypeWindsurf, err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")
	gitspaceLogger.Info(fmt.Sprintf("Successfully set up %s IDE inside container", enum.IDETypeWindsurf))
	return nil
}

// Run starts the SSH server inside the container.
func (w *Windsurf) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Starting ssh-server...")
	err := runSSHServer(ctx, exec, w.config.Port, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run %s IDE: %w", enum.IDETypeWindsurf, err)
	}
	gitspaceLogger.Info("Successfully run ssh-server")
	return nil
}

// Port returns the port on which the ssh-server is listening.
func (w *Windsurf) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     w.config.Port,
		Protocol: enum.CommunicationProtocolSSH,
	}
}

// Type returns the IDE type this service represents.
func (w *Windsurf) Type() enum.IDEType {
	return enum.IDETypeWindsurf
}

// GenerateURL returns a ssh command needed to ssh details need to be pasted in windsurf to connect via remote ssh
// plugin.
func (w *Windsurf) GenerateURL(absoluteRepoPath, host, port, user string) string { //nolint:revive // match interface
	return fmt.Sprintf("%s@%s:%s", user, host, port)
}
