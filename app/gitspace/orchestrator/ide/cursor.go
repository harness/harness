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

var _ IDE = (*Cursor)(nil)

// CursorConfig defines Cursor IDE specific configuration.
type CursorConfig struct {
	Port int
}

// Cursor implements the IDE interface for Cursor IDE (VSCode-based).
type Cursor struct {
	config *CursorConfig
}

// NewCursorService creates a new Cursor IDE service.
func NewCursorService(config *CursorConfig) *Cursor {
	return &Cursor{config: config}
}

// Setup installs the SSH server inside the container.
func (c *Cursor) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container...")
	err := setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", enum.IDETypeCursor, err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")
	gitspaceLogger.Info(fmt.Sprintf("Successfully set up %s IDE inside container", enum.IDETypeCursor))
	return nil
}

// Run starts the SSH server inside the container.
func (c *Cursor) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Starting ssh-server...")
	err := runSSHServer(ctx, exec, c.config.Port, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run %s IDE: %w", enum.IDETypeCursor, err)
	}
	gitspaceLogger.Info("Successfully run ssh-server")
	return nil
}

// Port returns the port on which the ssh-server is listening.
func (c *Cursor) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     c.config.Port,
		Protocol: enum.CommunicationProtocolSSH,
	}
}

// Type returns the IDE type this service represents.
func (c *Cursor) Type() enum.IDEType {
	return enum.IDETypeCursor
}

// GenerateURL returns SSH config snippet that user need to add to ssh config to connect with cursor.
func (c *Cursor) GenerateURL(_, host, port, user string) string { //nolint:revive // match interface
	return fmt.Sprintf(`
Host gitspace-%s-%s
  HostName %s
  Port %s
  User %s
  StrictHostKeyChecking no`,
		host, port,
		host,
		port,
		user,
	)
}
